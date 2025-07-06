from collections.abc import Mapping
from typing import TYPE_CHECKING, Any, TypeVar, Union

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

if TYPE_CHECKING:
    from ..models.job_status import JobStatus


T = TypeVar("T", bound="JobResponse")


@_attrs_define
class JobResponse:
    """
    Attributes:
        jobs (Union[Unset, list['JobStatus']]):
        page (Union[Unset, int]):
        page_size (Union[Unset, int]):
        total_jobs (Union[Unset, int]):
        total_pages (Union[Unset, int]):
    """

    jobs: Union[Unset, list["JobStatus"]] = UNSET
    page: Union[Unset, int] = UNSET
    page_size: Union[Unset, int] = UNSET
    total_jobs: Union[Unset, int] = UNSET
    total_pages: Union[Unset, int] = UNSET
    additional_properties: dict[str, Any] = _attrs_field(init=False, factory=dict)

    def to_dict(self) -> dict[str, Any]:
        jobs: Union[Unset, list[dict[str, Any]]] = UNSET
        if not isinstance(self.jobs, Unset):
            jobs = []
            for jobs_item_data in self.jobs:
                jobs_item = jobs_item_data.to_dict()
                jobs.append(jobs_item)

        page = self.page

        page_size = self.page_size

        total_jobs = self.total_jobs

        total_pages = self.total_pages

        field_dict: dict[str, Any] = {}
        field_dict.update(self.additional_properties)
        field_dict.update({})
        if jobs is not UNSET:
            field_dict["jobs"] = jobs
        if page is not UNSET:
            field_dict["page"] = page
        if page_size is not UNSET:
            field_dict["pageSize"] = page_size
        if total_jobs is not UNSET:
            field_dict["totalJobs"] = total_jobs
        if total_pages is not UNSET:
            field_dict["totalPages"] = total_pages

        return field_dict

    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.job_status import JobStatus

        d = dict(src_dict)
        jobs = []
        _jobs = d.pop("jobs", UNSET)
        for jobs_item_data in _jobs or []:
            jobs_item = JobStatus.from_dict(jobs_item_data)

            jobs.append(jobs_item)

        page = d.pop("page", UNSET)

        page_size = d.pop("pageSize", UNSET)

        total_jobs = d.pop("totalJobs", UNSET)

        total_pages = d.pop("totalPages", UNSET)

        job_response = cls(
            jobs=jobs,
            page=page,
            page_size=page_size,
            total_jobs=total_jobs,
            total_pages=total_pages,
        )

        job_response.additional_properties = d
        return job_response

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
