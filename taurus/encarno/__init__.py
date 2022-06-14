import http
import json
import logging
import os.path
import platform
import struct
import traceback
from json import JSONDecodeError
from urllib.parse import urlencode, urlparse

import bzt.engine
import yaml
from bzt import ToolError, TaurusInternalException, TaurusConfigError
from bzt.engine import ScenarioExecutor, HavingInstallableTools, Scenario
from bzt.modules import ExecutorWidget
from bzt.modules.aggregator import ResultsReader, ConsolidatingAggregator
from bzt.requests_model import HTTPRequest
from bzt.utils import RequiredTool, FileReader, shutdown_process, BetterDict, dehumanize_time, is_windows, is_mac, \
    ProgressBarContext, ExceptionalDownloader
from bzt.utils import get_full_path, CALL_PROBLEMS


class EncarnoExecutor(ScenarioExecutor, HavingInstallableTools):
    def __init__(self):
        super().__init__()
        self.waiting_warning_cnt = 0
        self.tool = None
        self.process = None
        self.generator = None

    def prepare(self):
        super().prepare()
        self.install_required_tools()
        self.stdout = open(self.engine.create_artifact("encarno", ".out"), 'w')
        self.stderr = open(self.engine.create_artifact("encarno", ".err"), 'w')

        self.generator = EncarnoFilesGenerator(self, self.log)
        self.generator.generate_payload(self.get_scenario())
        self.generator.generate_config(self.get_scenario(), self.get_load())

        self.reader = self.generator.get_results_reader()
        if isinstance(self.engine.aggregator, ConsolidatingAggregator):
            self.engine.aggregator.add_underling(self.reader)

    def install_required_tools(self):
        self.tool = self._get_tool(ToolBinary, config=self.settings)

        if not self.tool.check_if_installed():
            self.tool.install()

    def startup(self):
        cmdline = [self.tool.tool_path, self.generator.config_file]
        self.process = self._execute(cmdline)

    def check(self):
        retcode = self.process.poll()
        if retcode is not None:
            if retcode != 0:
                raise ToolError("%s exit code: %s" % (self.tool, retcode), self.get_error_diagnostics())
            return True

        waiting = self.reader.health_reader.cnt_waiting
        sleeping = self.reader.health_reader.cnt_sleeping
        lag = self.reader.health_reader.lag
        if waiting > 0:
            self.waiting_warning_cnt += 1
            if self.waiting_warning_cnt >= 3:
                self.log.warning("Encarno has %d workers waiting for inputs. Is load generator overloaded?" % waiting)
        else:
            self.waiting_warning_cnt = 0

        if self.widget:
            label = ["%r: " % self, ("graph fail" if waiting > 0 else "stat-txt", "%d wait" % waiting), ]
            label += [", ", "%d busy" % self.reader.health_reader.cnt_busy, ]
            if self.get_load().throughput:
                label += [
                    ", ", ("graph vc" if sleeping == 0 else "stat-txt", "%d sleep" % sleeping),
                    ", ", ("graph fail" if lag != "0s" else "stat-txt", "%s lag" % lag),
                ]
            self.widget.widgets[0].set_text(label)

        return False

    def shutdown(self):
        shutdown_process(self.process, self.log)

    def get_error_diagnostics(self):
        diagnostics = []
        if self.generator is not None:
            if self.stdout is not None:
                with open(self.stdout.name) as fds:
                    contents = fds.read().strip()
                    if contents:
                        diagnostics.append("Tool STDOUT:\n" + contents)
            if self.stderr is not None:
                with open(self.stderr.name) as fds:
                    contents = fds.read().strip()
                    # TODO: find and focus on panic / error messages
                    if contents:
                        diagnostics.append("Tool STDERR:\n" + contents)
        return diagnostics

    def get_widget(self):
        """
        Add progress widget to console screen sidebar
        """
        if not self.widget:
            scen = self.execution.get("scenario")
            label = "Encarno: %s" % scen if isinstance(scen, str) else "..."  # TODO
            self.widget = ExecutorWidget(self, label)
        return self.widget

    def resource_files(self):
        script = self.get_script_path()
        if script:
            return [script]
        else:
            return []


class ToolBinary(RequiredTool):
    DOWNLOAD_LINK = "https://github.com/undera/encarno/releases/download/{version}/encarno-{platform}-{arch}"
    VERSION = "0.0"  # TODO: replace with automatic version
    LOCAL_PATH = "~/.bzt/encarno-taurus/{version}/"

    def __init__(self, config=None, **kwargs):
        settings = config or {}
        version = settings.get("version", self.VERSION)
        self.tool_path = get_full_path(settings.get("path", self.LOCAL_PATH.format(version=version) + '/encarno'))
        if is_windows():
            platf = 'windows'
        elif is_mac():
            platf = 'darwin'
        else:
            platf = 'linux'

        arch = "arm64" if "arm" in platform.processor() else "amd64"  # 32-bit dudes go build it from source

        download_link = settings.get("download-link", self.DOWNLOAD_LINK).format(version=version, platform=platf,
                                                                                 arch=arch)
        super().__init__(tool_path=self.tool_path, download_link=download_link, version=version, **kwargs)

    def check_if_installed(self):
        self.log.debug("Trying: %s", self.tool_path)
        try:
            out, err = self.call([self.tool_path, "-help"])
        except CALL_PROBLEMS as exc:
            self.log.info("Tool check failed: %s", exc)
            # if err:
            #    self.log.warning("Tool check stderr: %s", err)

            return False

        self.log.debug("Tool check stdout: %s", out)
        return True

    def install(self):
        if self.dry_install:
            self.log.info(f"Dry installation for {self.tool_name}")
            return

        if not self.installable:
            msg = "%s isn't found, automatic installation isn't implemented" % self.tool_name
            if self.mandatory:
                raise ToolError(msg)
            else:
                self.log.warning(msg)
                return

        with ProgressBarContext() as pbar:
            if not os.path.exists(os.path.dirname(self.tool_path)):
                os.makedirs(os.path.dirname(self.tool_path))
            downloader = ExceptionalDownloader(self.http_client)
            self.log.info("Downloading %s", self.download_link)
            downloader.get(self.download_link, self.tool_path, reporthook=pbar.download_callback)

            os.chmod(get_full_path(self.tool_path), 0o755)

            if self.check_if_installed():
                return self.tool_path
            else:
                raise ToolError("Unable to run %s after installation!" % self.tool_name)


class EncarnoFilesGenerator(object):
    def __init__(self, executor, base_logger):
        """
        :type executor: bzt.engine.modules.ScenarioExecutor
        :type base_logger: logging.Logger
        """
        super().__init__()
        self.output_strings = None
        self.input_strings = None
        self.output_format = "bin"
        self.stats_file = None
        self.kpi_file = None
        self.log = base_logger.getChild(self.__class__.__name__)
        self.executor = executor
        self.config_file = None
        self.engine = executor.engine
        self.settings = executor.settings
        self.execution = executor.execution
        self.payload_file = None

    def generate_config(self, scenario, load):
        self.stats_file = self.engine.create_artifact("encarno_health", ".ldjson")
        timeout = dehumanize_time(scenario.get("timeout", "10s"))

        trace_level = int(scenario.get("trace-level", self.settings.get("trace-level", "1000")))
        cfg = {
            "protocol": {
                "driver": scenario.get('protocol', 'http'),
                "timeout": "%ss" % timeout,
                "maxconnections": load.concurrency,
                "tlsconf": scenario.get("tls-config", self.settings.get("tls-config", {}))
            },
            "input": {
                "payloadfile": self.payload_file,
                "stringsfile": self.input_strings if self.input_strings else "",
                "iterationlimit": load.iterations,
            },
            "output": {
                "reqrespfile": self.engine.create_artifact("encarno_trace", ".txt") if trace_level < 1000 else "",
                "reqrespfilelevel": trace_level,
            },
            "workers": {
                "mode": "open" if load.throughput else "closed",
                "workloadschedule": self._translate_load(load),
                "maxworkers": load.concurrency,
            }
        }

        self.output_format = self.settings.get("output-format", self.output_format)
        self.kpi_file = self.engine.create_artifact("encarno_results", "." + self.output_format)
        if self.output_format == "bin":
            cfg['output']["binaryfile"] = self.kpi_file
            self.output_strings = self.engine.create_artifact("encarno_results", ".ostr")
            cfg['output']["stringsfile"] = self.output_strings
        elif self.output_format == "ldjson":
            cfg['output']["ldjsonfile"] = self.kpi_file
        else:
            raise TaurusConfigError("Unsupported output-format: %s" % self.output_format)

        self.config_file = self.engine.create_artifact("encarno_cfg", ".yaml")
        with open(self.config_file, "w") as fp:
            yaml.safe_dump(cfg, fp)

    def _translate_load(self, load):
        res = []
        level = load.throughput if load.throughput else load.concurrency
        if load.ramp_up:
            if load.steps:
                step_dur = (load.ramp_up / load.steps)
                for step in range(load.steps):
                    step_level = round(level * (step + 1) / load.steps)
                    res.append({
                        "levelstart": step_level,
                        "levelend": step_level,
                        "duration": "%.3fs" % step_dur
                    })
            else:
                res.append({
                    "levelstart": 0,
                    "levelend": level,
                    "duration": "%.3fs" % load.ramp_up
                })

        if load.hold:
            res.append({
                "levelstart": level,
                "levelend": level,
                "duration": "%.3fs" % load.hold
            })
        return res

    def get_results_reader(self):
        if self.output_format == "bin":
            return KPIReaderBinary(self.kpi_file, self.log, self.executor.stderr.name)
        elif self.output_format == "ldjson":
            return KPIReaderLDJSON(self.kpi_file, self.log, self.executor.stderr.name)
        else:
            raise TaurusConfigError("Unsupported output-format: %s" % self.output_format)

    def generate_payload(self, scenario: bzt.engine.Scenario):
        self.payload_file = self.executor.get_script_path()

        if not self.payload_file:  # generation from requests
            self.payload_file = self.engine.create_artifact("encarno", '.inp')
            if self.settings.get("index-input-strings", True):
                self.input_strings = self.engine.create_artifact("encarno", '.istr')
            self.log.info("Generating payload file: %s", self.payload_file)
            str_list = self._generate_payload_inner(scenario)  # raises if there is no requests

            if self.input_strings:
                with open(self.input_strings, 'w') as fp:
                    fp.writelines(x + "\n" for x in str_list)
        else:
            self.input_strings = scenario.get("input-strings")

    def _generate_payload_inner(self, scenario):
        str_list = []
        req_val = scenario.get("requests")
        if isinstance(req_val, str):
            requests = self._text_file_reader(req_val, scenario)
        else:
            requests = scenario.get_requests()

        num_requests = 0
        with open(self.payload_file, 'w') as fds:
            for request in requests:
                if not isinstance(request, HTTPRequest):
                    msg = "Payload generator doesn't support '%s' blocks, skipping"
                    self.log.warning(msg, request.NAME)
                    continue

                host, tcp_payload = self._build_request(request, scenario)

                if self.input_strings:
                    if host not in str_list:
                        str_list.append(host)

                    if request.label not in str_list:
                        str_list.append(request.label)

                    metadata = {
                        "plen": len(tcp_payload.encode('utf-8')),
                        "iaddr": str_list.index(host) + 1,
                        "ilbl": str_list.index(request.label) + 1
                    }
                else:
                    metadata = {
                        "plen": len(tcp_payload.encode('utf-8')),
                        "address": host,
                        "label": request.label,
                    }

                fds.write(json.dumps(metadata))  # metadata
                fds.write("\r\n")  # sep
                fds.write(tcp_payload)  # payload
                fds.write("\r\n")  # sep

                num_requests += 1

        if not num_requests:
            if scenario.get('protocol') == "dummy":
                self.log.info("Dummy test uses dummy scenario")
                scenario['requests'] = ["/"]
                return self._generate_payload_inner(scenario)

            raise TaurusInternalException("No requests were generated, check your 'requests' section presence")

        return str_list

    def _build_request(self, request: HTTPRequest, scenario: Scenario):
        host_url, netloc, path = self._get_request_path(request, scenario)
        payload = "%s %s HTTP/1.1\r\n" % (request.method, path)
        headers = BetterDict.from_dict({"Host": netloc})
        if not scenario.get("keepalive", True):
            headers.merge({"Connection": 'close'})  # HTTP/1.1 implies keep-alive by default

        body = ""
        if isinstance(request.body, dict):
            if request.method != "GET":
                body = urlencode(request.body)
        elif isinstance(request.body, str):
            body = request.body
        elif request.body:
            msg = "Cannot handle 'body' option of type %s: %s"
            raise TaurusConfigError(msg % (type(request.body), request.body))

        if body:
            headers.merge({"Content-Length": len(body)})

        headers.merge(scenario.get_headers())
        headers.merge(request.headers)
        for header, value in headers.items():
            payload += "%s: %s\r\n" % (header, value)
        payload += "\r\n%s" % (body,)
        return host_url, payload

    def _get_request_path(self, request, scenario):
        parsed_url = urlparse(request.url)

        path = parsed_url.path
        if parsed_url.query:
            path += "?" + parsed_url.query
        else:
            if request.method == "GET" and isinstance(request.body, dict):
                path += "?" + urlencode(request.body)

        if not parsed_url.netloc:
            parsed_url = urlparse(scenario.get("default-address", ""))

        hostname = parsed_url.scheme + "://" + parsed_url.netloc

        return hostname, parsed_url.netloc, path if len(path) else '/'

    def _text_file_reader(self, filename, scenario):
        with open(filename) as fp:
            for line in fp:
                line = line.strip()
                if not line:
                    continue

                parts = line.split(" ")

                req = {
                    "url": parts[1 if len(parts) > 1 else 0]
                }

                if len(parts) > 1:
                    req["label"] = parts[0]

                yield HTTPRequest(req, scenario, self.engine)


def ns2sec(val):
    return int(val) / 1000000000.0


class KPIReaderLDJSON(ResultsReader):
    """
    Class to read KPI from LDJSON file
    """

    def __init__(self, filename, parent_logger, health_filename):
        super().__init__()
        self.log = parent_logger.getChild(self.__class__.__name__)
        self.file = FileReader(filename=filename, parent_logger=self.log)
        self.partial_buffer = ""
        self.health_reader = HealthReader(health_filename, parent_logger)

    def _read(self, last_pass=False):
        """
        Generator method that returns next portion of data

        :type last_pass: bool
        """
        self.health_reader.read(last_pass)

        lines = self.file.get_lines(size=1024 * 1024, last_pass=last_pass)

        for line in lines:
            if not line.endswith("\n"):
                self.partial_buffer += line
                continue

            line = self.partial_buffer + line
            self.partial_buffer = ""

            try:
                row = json.loads(line)
            except JSONDecodeError:
                self.log.warning("Failed to decode JSON line: %s", traceback.format_exc())
                continue

            label = row["Label"]

            try:
                rtm = ns2sec(row["Elapsed"])
                ltc = ns2sec(row["FirstByteTime"])
                cnn = ns2sec(row["ConnectTime"])
                # NOTE: actually we have precise send and receive time here...
            except BaseException:
                raise ToolError("Reader: failed record: %s" % row)

            error = row["ErrorStr"] if row["ErrorStr"] else None
            rcd = str(row["Status"])

            if row["Status"] >= 400 and not error:  # TODO: should this be under config flag?
                error = http.HTTPStatus(row["Status"]).phrase

            tstmp = int(row["StartTS"])

            byte_count = row["SentBytesCount"] + row["RespBytesCount"]
            concur = row["Concurrency"]
            yield tstmp, label, concur, rtm, cnn, ltc, rcd, error, '', byte_count

    def _ramp_up_exclude(self):
        return False


class KPIReaderBinary(ResultsReader):
    """
    Class to read KPI from LDJSON file
    """
    FORMAT = "<Q HH L 5d LH QQ"
    CHUNK_LEN = struct.calcsize(FORMAT)

    def __init__(self, filename, parent_logger, health_filename):
        super().__init__()
        self.log = parent_logger.getChild(self.__class__.__name__)
        self.file = FileReader(filename=filename, file_opener=lambda x: open(x, 'rb'), parent_logger=self.log)
        self.partial_buffer = bytes()
        self.health_reader = HealthReader(health_filename, parent_logger)

    def _read(self, last_pass=False):
        """
        Generator method that returns next portion of data

        :type last_pass: bool
        """
        self.health_reader.read(last_pass)

        data = self.file.get_bytes(size=1024 * 1024, last_pass=last_pass, decode=False)
        if data is None:
            return  # file not ready yet

        data = self.partial_buffer + data

        while len(data) >= self.CHUNK_LEN:
            item = struct.unpack_from(self.FORMAT, data)
            data = data[self.CHUNK_LEN:]

            tstmp, rcd, err_idx, concur, rtm, cnn, sent, ltc, recv, wrk, lbl_idx, sbytes, rbytes = item

            error = None
            label = ""  # TODO

            if err_idx > 0:
                error = "some"  # TODO

            if rcd >= 400 and not error:  # TODO: should this be under config flag?
                error = http.HTTPStatus(rcd).phrase

            byte_count = sbytes + rbytes
            yield tstmp, label, concur, rtm, cnn, ltc, str(rcd), error, '', byte_count

        self.partial_buffer = data

    def _ramp_up_exclude(self):
        return False


class HealthReader:
    # waiting workers - our schedule is unable to generate fast enough
    # no sleeping - not enough workers

    def __init__(self, filename, parent_logger) -> None:
        super().__init__()
        self.lag = ""
        self.cnt_waiting = 0
        self.cnt_working = 0
        self.cnt_sleeping = 0
        self.cnt_busy = 0
        self.log = parent_logger.getChild(self.__class__.__name__)
        self.file = FileReader(filename=filename, parent_logger=self.log)
        self.partial_buffer = ""

    def read(self, last_pass=False):
        lines = self.file.get_lines(size=1024 * 1024, last_pass=last_pass)

        for line in lines:
            if not line.endswith("\n"):
                self.partial_buffer += line
                continue

            line = self.partial_buffer + line
            self.partial_buffer = ""

            if "Workers: " in line:
                try:
                    ts, _, line = line.partition(" ")
                    level, _, line = line.partition(" ")
                    parts = line.split(' ')
                    self.cnt_waiting = int(parts[2][:-1])
                    self.cnt_working = int(parts[4][:-1])
                    self.cnt_sleeping = int(parts[6][:-1])
                    self.cnt_busy = int(parts[8][:-1])
                    self.lag = parts[10][:-1]
                except KeyboardInterrupt:
                    raise
                except BaseException:
                    self.log.warning("Failed to parse encarno health line: %s", traceback.format_exc())
                    self.log.warning("The line was: %s", line)
