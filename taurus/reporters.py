import csv

from bzt.engine import Reporter
from bzt.modules.aggregator import AggregatorListener, ResultsProvider, DataPoint, KPISet


class CSVFile(Reporter, AggregatorListener):
    """
    :type writer: csv.DictWriter
    """

    def __init__(self):
        super().__init__()
        self.fp = None
        self.writer = None

    def prepare(self):
        super().prepare()
        fname = self.engine.create_artifact("aggregate", ".csv")
        self.log.info("Writing per-second CSV stats into: %s", fname)
        self.fp = open(fname, "w")
        self.writer = csv.DictWriter(self.fp, fieldnames=["ts", "conc", "succ", "fail", "rt"], dialect="excel")
        self.writer.writeheader()

        if isinstance(self.engine.aggregator, ResultsProvider):
            self.engine.aggregator.add_listener(self)

    def finalize(self):
        if self.fp:
            self.fp.close()

    def aggregated_second(self, data: DataPoint):
        self.writer.writerow({
            "ts": data[DataPoint.TIMESTAMP],
            "conc": data[DataPoint.CURRENT][''][KPISet.CONCURRENCY],
            "succ": data[DataPoint.CURRENT][''][KPISet.SUCCESSES],
            "fail": data[DataPoint.CURRENT][''][KPISet.FAILURES],
            "rt": data[DataPoint.CURRENT][''][KPISet.AVG_RESP_TIME],
        })
