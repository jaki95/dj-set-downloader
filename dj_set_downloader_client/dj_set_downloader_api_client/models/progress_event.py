from collections.abc import Mapping
from typing import TYPE_CHECKING, Any, TypeVar, Union, cast

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..models.progress_stage import ProgressStage
from ..types import UNSET, Unset

if TYPE_CHECKING:
    from ..models.progress_track_details import ProgressTrackDetails


T = TypeVar("T", bound="ProgressEvent")


@_attrs_define
class ProgressEvent:
    """
    Attributes:
        data (Union[Unset, list[int]]):
        error (Union[Unset, str]):
        message (Union[Unset, str]):
        progress (Union[Unset, float]):
        stage (Union[Unset, ProgressStage]):
        timestamp (Union[Unset, str]):
        track_details (Union[Unset, ProgressTrackDetails]):
    """

    data: Union[Unset, list[int]] = UNSET
    error: Union[Unset, str] = UNSET
    message: Union[Unset, str] = UNSET
    progress: Union[Unset, float] = UNSET
    stage: Union[Unset, ProgressStage] = UNSET
    timestamp: Union[Unset, str] = UNSET
    track_details: Union[Unset, "ProgressTrackDetails"] = UNSET
    additional_properties: dict[str, Any] = _attrs_field(init=False, factory=dict)

    def to_dict(self) -> dict[str, Any]:
        data: Union[Unset, list[int]] = UNSET
        if not isinstance(self.data, Unset):
            data = self.data

        error = self.error

        message = self.message

        progress = self.progress

        stage: Union[Unset, str] = UNSET
        if not isinstance(self.stage, Unset):
            stage = self.stage.value

        timestamp = self.timestamp

        track_details: Union[Unset, dict[str, Any]] = UNSET
        if not isinstance(self.track_details, Unset):
            track_details = self.track_details.to_dict()

        field_dict: dict[str, Any] = {}
        field_dict.update(self.additional_properties)
        field_dict.update({})
        if data is not UNSET:
            field_dict["data"] = data
        if error is not UNSET:
            field_dict["error"] = error
        if message is not UNSET:
            field_dict["message"] = message
        if progress is not UNSET:
            field_dict["progress"] = progress
        if stage is not UNSET:
            field_dict["stage"] = stage
        if timestamp is not UNSET:
            field_dict["timestamp"] = timestamp
        if track_details is not UNSET:
            field_dict["trackDetails"] = track_details

        return field_dict

    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.progress_track_details import ProgressTrackDetails

        d = dict(src_dict)
        data = cast(list[int], d.pop("data", UNSET))

        error = d.pop("error", UNSET)

        message = d.pop("message", UNSET)

        progress = d.pop("progress", UNSET)

        _stage = d.pop("stage", UNSET)
        stage: Union[Unset, ProgressStage]
        if isinstance(_stage, Unset):
            stage = UNSET
        else:
            stage = ProgressStage(_stage)

        timestamp = d.pop("timestamp", UNSET)

        _track_details = d.pop("trackDetails", UNSET)
        track_details: Union[Unset, ProgressTrackDetails]
        if isinstance(_track_details, Unset):
            track_details = UNSET
        else:
            track_details = ProgressTrackDetails.from_dict(_track_details)

        progress_event = cls(
            data=data,
            error=error,
            message=message,
            progress=progress,
            stage=stage,
            timestamp=timestamp,
            track_details=track_details,
        )

        progress_event.additional_properties = d
        return progress_event

    @property
    def additional_keys(self) -> list[str]:
        return list(self.additional_properties.keys())

    def __getitem__(self, key: str) -> Any:
        return self.additional_properties[key]

    def __setitem__(self, key: str, value: Any) -> None:
        self.additional_properties[key] = value

    def __delitem__(self, key: str) -> None:
        del self.additional_properties[key]

    def __contains__(self, key: str) -> bool:
        return key in self.additional_properties
