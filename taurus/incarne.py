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

    def prepare(self):
        super().prepare()
        self.install_required_tools()

    def install_required_tools(self):
        self.tool = self._get_tool(IncarneBinary, config=self.settings)

        if not self.tool.check_if_installed():
            self.tool.install()


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
            return False

        self.log.debug("Tool check stdout: %s", out)
        if err:
            self.log.warning("Tool check stderr: %s", err)
        return True
