from collections.abc import Mapping
from typing import Any, TypeVar, Union

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

T = TypeVar("T", bound="JobRequest")


@_attrs_define
class JobRequest:
    """
    Attributes:
        tracklist (str):
        url (str):
        file_extension (Union[Unset, str]):
        max_concurrent_tasks (Union[Unset, int]):
    """

    tracklist: str
    url: str
    file_extension: Union[Unset, str] = UNSET
    max_concurrent_tasks: Union[Unset, int] = UNSET
    additional_properties: dict[str, Any] = _attrs_field(init=False, factory=dict)

    def to_dict(self) -> dict[str, Any]:
        tracklist = self.tracklist

        url = self.url

        file_extension = self.file_extension

        max_concurrent_tasks = self.max_concurrent_tasks

        field_dict: dict[str, Any] = {}
        field_dict.update(self.additional_properties)
        field_dict.update(
            {
                "tracklist": tracklist,
                "url": url,
            }
        )
        if file_extension is not UNSET:
            field_dict["fileExtension"] = file_extension
        if max_concurrent_tasks is not UNSET:
            field_dict["maxConcurrentTasks"] = max_concurrent_tasks

        return field_dict

    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        tracklist = d.pop("tracklist")

        url = d.pop("url")

        file_extension = d.pop("fileExtension", UNSET)

        max_concurrent_tasks = d.pop("maxConcurrentTasks", UNSET)

        job_request = cls(
            tracklist=tracklist,
            url=url,
            file_extension=file_extension,
            max_concurrent_tasks=max_concurrent_tasks,
        )

        job_request.additional_properties = d
        return job_request

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
