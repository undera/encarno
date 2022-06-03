import csv
import json
import traceback
from json import JSONDecodeError
from os import strerror

import yaml
from bzt import ToolError
from bzt.engine import ScenarioExecutor, HavingInstallableTools
from bzt.modules.aggregator import ResultsReader, DataPoint, KPISet, ConsolidatingAggregator
from bzt.utils import RequiredTool, FileReader, shutdown_process
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

    def _read(self, last_pass=False):
        """
        Generator method that returns next portion of data

        :type last_pass: bool
        """

        self.stats_reader.read_file()

        lines = self.file.get_lines(size=1024 * 1024, last_pass=last_pass)

        for line in lines:
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

            error = row["Error"]
            rcd = str(row["Status"])

            tstmp = int(isoparse(row["StartTime"]).timestamp())

            byte_count = row["SentBytesCount"] + row["RespBytesCount"]
            concur = 0
            yield tstmp, label, concur, rtm, cnn, ltc, rcd, error, '', byte_count

    def _calculate_datapoints(self, final_pass=False):
        for point in super()._calculate_datapoints(final_pass):
            concurrency = self.stats_reader.get_data(point[DataPoint.TIMESTAMP])

            for label_data in point[DataPoint.CURRENT].values():
                label_data[KPISet.CONCURRENCY] = concurrency

            yield point


class IncarneHealthReader(object):
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
            self.log.debug("No active instances info for %s", tstmp)
            return self.last_data
