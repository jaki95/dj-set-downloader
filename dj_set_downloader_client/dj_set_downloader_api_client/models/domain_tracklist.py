from collections.abc import Mapping
from typing import TYPE_CHECKING, Any, TypeVar, Union

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

if TYPE_CHECKING:
    from ..models.domain_track import DomainTrack


T = TypeVar("T", bound="DomainTracklist")


@_attrs_define
class DomainTracklist:
    """
    Attributes:
        artist (Union[Unset, str]):
        genre (Union[Unset, str]):
        name (Union[Unset, str]):
        tracks (Union[Unset, list['DomainTrack']]):
        year (Union[Unset, int]):
    """

    artist: Union[Unset, str] = UNSET
    genre: Union[Unset, str] = UNSET
    name: Union[Unset, str] = UNSET
    tracks: Union[Unset, list["DomainTrack"]] = UNSET
    year: Union[Unset, int] = UNSET
    additional_properties: dict[str, Any] = _attrs_field(init=False, factory=dict)

    def to_dict(self) -> dict[str, Any]:
        artist = self.artist

        genre = self.genre

        name = self.name

        tracks: Union[Unset, list[dict[str, Any]]] = UNSET
        if not isinstance(self.tracks, Unset):
            tracks = []
            for tracks_item_data in self.tracks:
                tracks_item = tracks_item_data.to_dict()
                tracks.append(tracks_item)

        year = self.year

        field_dict: dict[str, Any] = {}
        field_dict.update(self.additional_properties)
        field_dict.update({})
        if artist is not UNSET:
            field_dict["artist"] = artist
        if genre is not UNSET:
            field_dict["genre"] = genre
        if name is not UNSET:
            field_dict["name"] = name
        if tracks is not UNSET:
            field_dict["tracks"] = tracks
        if year is not UNSET:
            field_dict["year"] = year

        return field_dict

    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.domain_track import DomainTrack

        d = dict(src_dict)
        artist = d.pop("artist", UNSET)

        genre = d.pop("genre", UNSET)

        name = d.pop("name", UNSET)

        tracks = []
        _tracks = d.pop("tracks", UNSET)
        for tracks_item_data in _tracks or []:
            tracks_item = DomainTrack.from_dict(tracks_item_data)

            tracks.append(tracks_item)

        year = d.pop("year", UNSET)

        domain_tracklist = cls(
            artist=artist,
            genre=genre,
            name=name,
            tracks=tracks,
            year=year,
        )

        domain_tracklist.additional_properties = d
        return domain_tracklist

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
