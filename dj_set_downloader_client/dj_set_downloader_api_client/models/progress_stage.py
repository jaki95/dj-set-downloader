from enum import Enum


class ProgressStage(str, Enum):
    COMPLETE = "complete"
    DOWNLOADING = "downloading"
    ERROR = "error"
    IMPORTING = "importing"
    INITIALIZING = "initializing"
    PROCESSING = "processing"

    def __str__(self) -> str:
        return str(self.value)
