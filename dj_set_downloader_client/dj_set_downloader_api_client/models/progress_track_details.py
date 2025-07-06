from collections.abc import Mapping
from typing import Any, TypeVar, Union

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

T = TypeVar("T", bound="ProgressTrackDetails")


@_attrs_define
class ProgressTrackDetails:
    """
    Attributes:
        current_track (Union[Unset, str]):
        processed_tracks (Union[Unset, int]):
        total_tracks (Union[Unset, int]):
        track_number (Union[Unset, int]):
    """

    current_track: Union[Unset, str] = UNSET
    processed_tracks: Union[Unset, int] = UNSET
    total_tracks: Union[Unset, int] = UNSET
    track_number: Union[Unset, int] = UNSET
    additional_properties: dict[str, Any] = _attrs_field(init=False, factory=dict)

    def to_dict(self) -> dict[str, Any]:
        current_track = self.current_track

        processed_tracks = self.processed_tracks

        total_tracks = self.total_tracks

        track_number = self.track_number

        field_dict: dict[str, Any] = {}
        field_dict.update(self.additional_properties)
        field_dict.update({})
        if current_track is not UNSET:
            field_dict["currentTrack"] = current_track
        if processed_tracks is not UNSET:
            field_dict["processedTracks"] = processed_tracks
        if total_tracks is not UNSET:
            field_dict["totalTracks"] = total_tracks
        if track_number is not UNSET:
            field_dict["trackNumber"] = track_number

        return field_dict

    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        current_track = d.pop("currentTrack", UNSET)

        processed_tracks = d.pop("processedTracks", UNSET)

        total_tracks = d.pop("totalTracks", UNSET)

        track_number = d.pop("trackNumber", UNSET)

        progress_track_details = cls(
            current_track=current_track,
            processed_tracks=processed_tracks,
            total_tracks=total_tracks,
            track_number=track_number,
        )

        progress_track_details.additional_properties = d
        return progress_track_details

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
