from collections.abc import Mapping
from typing import Any, TypeVar, Union

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

T = TypeVar("T", bound="DomainTrack")


@_attrs_define
class DomainTrack:
    """
    Attributes:
        artist (Union[Unset, str]):
        end_time (Union[Unset, str]):
        name (Union[Unset, str]):
        start_time (Union[Unset, str]):
        track_number (Union[Unset, int]):
    """

    artist: Union[Unset, str] = UNSET
    end_time: Union[Unset, str] = UNSET
    name: Union[Unset, str] = UNSET
    start_time: Union[Unset, str] = UNSET
    track_number: Union[Unset, int] = UNSET
    additional_properties: dict[str, Any] = _attrs_field(init=False, factory=dict)

    def to_dict(self) -> dict[str, Any]:
        artist = self.artist

        end_time = self.end_time

        name = self.name

        start_time = self.start_time

        track_number = self.track_number

        field_dict: dict[str, Any] = {}
        field_dict.update(self.additional_properties)
        field_dict.update({})
        if artist is not UNSET:
            field_dict["artist"] = artist
        if end_time is not UNSET:
            field_dict["end_time"] = end_time
        if name is not UNSET:
            field_dict["name"] = name
        if start_time is not UNSET:
            field_dict["start_time"] = start_time
        if track_number is not UNSET:
            field_dict["track_number"] = track_number

        return field_dict

    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        artist = d.pop("artist", UNSET)

        end_time = d.pop("end_time", UNSET)

        name = d.pop("name", UNSET)

        start_time = d.pop("start_time", UNSET)

        track_number = d.pop("track_number", UNSET)

        domain_track = cls(
            artist=artist,
            end_time=end_time,
            name=name,
            start_time=start_time,
            track_number=track_number,
        )

        domain_track.additional_properties = d
        return domain_track

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
