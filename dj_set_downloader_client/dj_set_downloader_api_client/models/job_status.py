from collections.abc import Mapping
from typing import TYPE_CHECKING, Any, TypeVar, Union, cast

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

if TYPE_CHECKING:
    from ..models.domain_tracklist import DomainTracklist
    from ..models.progress_event import ProgressEvent


T = TypeVar("T", bound="JobStatus")


@_attrs_define
class JobStatus:
    """
    Attributes:
        end_time (Union[Unset, str]):
        error (Union[Unset, str]):
        events (Union[Unset, list['ProgressEvent']]):
        id (Union[Unset, str]):
        message (Union[Unset, str]):
        progress (Union[Unset, float]):
        results (Union[Unset, list[str]]):
        start_time (Union[Unset, str]):
        status (Union[Unset, str]):
        tracklist (Union[Unset, DomainTracklist]):
    """

    end_time: Union[Unset, str] = UNSET
    error: Union[Unset, str] = UNSET
    events: Union[Unset, list["ProgressEvent"]] = UNSET
    id: Union[Unset, str] = UNSET
    message: Union[Unset, str] = UNSET
    progress: Union[Unset, float] = UNSET
    results: Union[Unset, list[str]] = UNSET
    start_time: Union[Unset, str] = UNSET
    status: Union[Unset, str] = UNSET
    tracklist: Union[Unset, "DomainTracklist"] = UNSET
    additional_properties: dict[str, Any] = _attrs_field(init=False, factory=dict)

    def to_dict(self) -> dict[str, Any]:
        end_time = self.end_time

        error = self.error

        events: Union[Unset, list[dict[str, Any]]] = UNSET
        if not isinstance(self.events, Unset):
            events = []
            for events_item_data in self.events:
                events_item = events_item_data.to_dict()
                events.append(events_item)

        id = self.id

        message = self.message

        progress = self.progress

        results: Union[Unset, list[str]] = UNSET
        if not isinstance(self.results, Unset):
            results = self.results

        start_time = self.start_time

        status = self.status

        tracklist: Union[Unset, dict[str, Any]] = UNSET
        if not isinstance(self.tracklist, Unset):
            tracklist = self.tracklist.to_dict()

        field_dict: dict[str, Any] = {}
        field_dict.update(self.additional_properties)
        field_dict.update({})
        if end_time is not UNSET:
            field_dict["endTime"] = end_time
        if error is not UNSET:
            field_dict["error"] = error
        if events is not UNSET:
            field_dict["events"] = events
        if id is not UNSET:
            field_dict["id"] = id
        if message is not UNSET:
            field_dict["message"] = message
        if progress is not UNSET:
            field_dict["progress"] = progress
        if results is not UNSET:
            field_dict["results"] = results
        if start_time is not UNSET:
            field_dict["startTime"] = start_time
        if status is not UNSET:
            field_dict["status"] = status
        if tracklist is not UNSET:
            field_dict["tracklist"] = tracklist

        return field_dict

    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.domain_tracklist import DomainTracklist
        from ..models.progress_event import ProgressEvent

        d = dict(src_dict)
        end_time = d.pop("endTime", UNSET)

        error = d.pop("error", UNSET)

        events = []
        _events = d.pop("events", UNSET)
        for events_item_data in _events or []:
            events_item = ProgressEvent.from_dict(events_item_data)

            events.append(events_item)

        id = d.pop("id", UNSET)

        message = d.pop("message", UNSET)

        progress = d.pop("progress", UNSET)

        results = cast(list[str], d.pop("results", UNSET))

        start_time = d.pop("startTime", UNSET)

        status = d.pop("status", UNSET)

        _tracklist = d.pop("tracklist", UNSET)
        tracklist: Union[Unset, DomainTracklist]
        if isinstance(_tracklist, Unset):
            tracklist = UNSET
        else:
            tracklist = DomainTracklist.from_dict(_tracklist)

        job_status = cls(
            end_time=end_time,
            error=error,
            events=events,
            id=id,
            message=message,
            progress=progress,
            results=results,
            start_time=start_time,
            status=status,
            tracklist=tracklist,
        )

        job_status.additional_properties = d
        return job_status

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
