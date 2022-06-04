import csv
import json
import traceback
from json import JSONDecodeError
from os import strerror
from urllib.parse import urlencode

import yaml
from bzt import ToolError, TaurusInternalException, TaurusConfigError
from bzt.engine import ScenarioExecutor, HavingInstallableTools
from bzt.modules import ExecutorWidget
from bzt.modules.aggregator import ResultsReader, DataPoint, KPISet, ConsolidatingAggregator
from bzt.requests_model import HTTPRequest
from bzt.utils import RequiredTool, FileReader, shutdown_process, BetterDict
from bzt.utils import get_full_path, CALL_PROBLEMS
from dateutil.parser import isoparse


class IncarneExecutor(ScenarioExecutor, HavingInstallableTools):
    def __init__(self):
        super().__init__()
        self.tool = None
        self.process = None
        self.generator = None

    def prepare(self):
        super().prepare()
        self.install_required_tools()
        self.stdout = open(self.engine.create_artifact("incarne", ".out"), 'w')
        self.stderr = open(self.engine.create_artifact("incarne", ".err"), 'w')
        self.generator = IncarneFilesGenerator(self, self.log)
        self.generator.generate_config(self.get_scenario(), self.get_load())
        self.generator.generate_payload(self.get_scenario())

        self.reader = self.generator.get_results_reader()
        if isinstance(self.engine.aggregator, ConsolidatingAggregator):
            self.engine.aggregator.add_underling(self.reader)

    def install_required_tools(self):
        self.tool = self._get_tool(IncarneBinary, config=self.settings)

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
            label = "Incarne: %s" % scen if isinstance(scen, str) else "..."  # TODO
            self.widget = ExecutorWidget(self, label)
        return self.widget

    def resource_files(self):
        script = self.get_script_path()
        if script:
            return [script]
        else:
            return []


class IncarneBinary(RequiredTool):
    def __init__(self, config=None, **kwargs):
        settings = config or {}

        # don't extend system-wide default
        tool_path = get_full_path(settings.get("path"), default="incarne")

        super().__init__(tool_path=tool_path, installable=False, **kwargs)

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


class IncarneFilesGenerator(object):
    def __init__(self, executor, base_logger):
        """
        :type executor: bzt.engine.modules.ScenarioExecutor
        :type base_logger: logging.Logger
        """
        super().__init__()
        self.stats_file = None
        self.kpi_file = None
        self.log = base_logger.getChild(self.__class__.__name__)
        self.executor = executor
        self.config_file = None
        self.engine = executor.engine
        self.settings = executor.settings
        self.execution = executor.execution

    def generate_config(self, scenario, load):
        self.kpi_file = self.engine.create_artifact("incarne_results", ".ldjson")
        self.stats_file = self.engine.create_artifact("incarne_health", ".ldjson")
        cfg = {
            "protocol": {
                "driver": scenario.get('protocol', 'http')
            },
            "input": {

            },
            "output": {
                "ldjsonfile": self.kpi_file
            },
            "workers": {
                "mode": "open" if load.throughput else "closed",
                "workloadschedule": self._translate_load(load)
            }
        }

        self.config_file = self.engine.create_artifact("incarne_cfg", ".yaml")
        with open(self.config_file, "w") as fp:
            yaml.safe_dump(cfg, fp)

    def _translate_load(self, load):
        res = []
        level = load.throughput if load.throughput else load.concurrency
        if load.ramp_up:
            if load.steps:
                pass  # TODO
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
        return IncarneKPIReader(self.kpi_file, self.log, self.stats_file)

    def generate_payload(self, scenario):
        self.payload_file = self.executor.get_script_path()

        if not self.payload_file:  # generation from requests
            self.payload_file = self.engine.create_artifact("pbench", '.src')
            self.log.info("Generating payload file: %s", self.payload_file)
            self._generate_payload_inner(scenario)  # raises if there is no requests

    def _generate_payload_inner(self, scenario):
        requests = scenario.get_requests()
        num_requests = 0
        with open(self.payload_file, 'w') as fds:
            for request in requests:
                if not isinstance(request, HTTPRequest):
                    msg = "PBench payload generator doesn't support '%s' blocks, skipping"
                    self.log.warning(msg, request.NAME)
                    continue

                http = self._build_request(request, scenario)
                fds.write("%s %s\r\n%s\r\n" % (len(http), request.label.replace(' ', '_'), http))
                num_requests += 1

        if not num_requests:
            raise TaurusInternalException("No requests were generated, check your 'requests' section presence")

    def _build_request(self, request, scenario):
        path = self._get_request_path(request, scenario)
        http = "%s %s HTTP/1.1\r\n" % (request.method, path)
        headers = BetterDict.from_dict({"Host": self.hostname})
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
            http += "%s: %s\r\n" % (header, value)
        http += "\r\n%s" % (body,)
        return http

    def _get_request_path(self, request, scenario):

        parsed_url = urlparse(request.url)

        if not self._target.get("scheme"):
            self._target["scheme"] = parsed_url.scheme

        if not self._target.get("netloc"):
            self._target["netloc"] = parsed_url.netloc

        if parsed_url.scheme != self._target["scheme"] or parsed_url.netloc != self._target["netloc"]:
            raise TaurusConfigError("Address port and host must be the same")
        path = parsed_url.path
        if parsed_url.query:
            path += "?" + parsed_url.query
        else:
            if request.method == "GET" and isinstance(request.body, dict):
                path += "?" + urlencode(request.body)
        if not parsed_url.netloc:
            parsed_url = parse.urlparse(scenario.get("default-address", ""))

        self.hostname = parsed_url.netloc.split(':')[0] if ':' in parsed_url.netloc else parsed_url.netloc
        self.use_ssl = parsed_url.scheme == 'https'
        if parsed_url.port:
            self.port = parsed_url.port
        else:
            self.port = 443 if self.use_ssl else 80

        return path if len(path) else '/'


def ns2sec(val):
    return int(val) / 1000000000.0


class IncarneKPIReader(ResultsReader):
    """
    Class to read KPI
    """

    def __init__(self, filename, parent_logger, health_filename):
        super().__init__()
        self.log = parent_logger.getChild(self.__class__.__name__)
        self.file = FileReader(filename=filename, parent_logger=self.log)
        self.stats_reader = IncarneHealthReader(health_filename, parent_logger)
        self.partial_buffer = ""

    def _read(self, last_pass=False):
        """
        Generator method that returns next portion of data

        :type last_pass: bool
        """

        self.stats_reader.read_file()

        lines = self.file.get_lines(size=1024 * 1024, last_pass=last_pass)

        for line in lines:
            if not line.endswith("\n"):
                self.partial_buffer += line
                continue

            line = "%s%s" % (self.partial_buffer, line)
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

            tstmp = int(row["StartTS"])

            byte_count = row["SentBytesCount"] + row["RespBytesCount"]
            concur = row["Concurrency"]
            yield tstmp, label, concur, rtm, cnn, ltc, rcd, error, '', byte_count

    def _calculate_datapoints(self, final_pass=False):
        for point in super()._calculate_datapoints(final_pass):
            concurrency = self.stats_reader.get_data(point[DataPoint.TIMESTAMP])

            for label_data in point[DataPoint.CURRENT].values():
                label_data[KPISet.CONCURRENCY] = concurrency

            yield point


class IncarneHealthReader(object):  # TODO: do we need it?
    def __init__(self, filename, parent_logger):
        super().__init__()
        self.log = parent_logger.getChild(self.__class__.__name__)
        self.file = FileReader(filename=filename, parent_logger=self.log)
        self.buffer = ''
        self.data = {}
        self.last_data = 0

    def read_file(self):
        pass

    def get_data(self, tstmp):
        if tstmp in self.data:
            self.last_data = self.data[tstmp]
            return self.data[tstmp]
        else:
            # self.log.debug("No active instances info for %s", tstmp)
            return self.last_data