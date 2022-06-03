import csv
import datetime
import json
import math
import os
import socket
import string
import struct
import time
from abc import abstractmethod
from os import strerror

import psutil
import yaml

from bzt import TaurusConfigError, ToolError, TaurusInternalException
from bzt.engine import ScenarioExecutor, HavingInstallableTools
from bzt.modules.aggregator import ResultsReader, DataPoint, KPISet, ConsolidatingAggregator
from bzt.modules.console import ExecutorWidget
from bzt.requests_model import HTTPRequest
from bzt.utils import RequiredTool, IncrementableProgressBar, FileReader, RESOURCES_DIR
from bzt.utils import shutdown_process, BetterDict, dehumanize_time, get_full_path, CALL_PROBLEMS


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
        self.log = base_logger.getChild(self.__class__.__name__)
        self.executor = executor
        self.config_file = None
        self.engine = executor.engine
        self.settings = executor.settings
        self.execution = executor.execution

    def generate_config(self, scenario, load):
        cfg = {
            "protocol": {
                "driver": scenario.get('protocol', 'http')
            },
            "input": {

            },
            "output": {
                "ldjsonfile": self.engine.create_artifact("incarne_results", ".ldjson")
            },
            "workers": {
                "mode": "open" if load.throughput else "closed",
                "workloadschedule": []
            }
        }

        self.config_file = self.engine.create_artifact("incarne_cfg", ".yaml")
        with open(self.config_file, "w") as fp:
            yaml.safe_dump(cfg, fp)
