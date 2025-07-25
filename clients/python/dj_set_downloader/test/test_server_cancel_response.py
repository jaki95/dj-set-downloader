# coding: utf-8

"""
    DJ Set Downloader API

    A REST API for downloading and processing DJ sets, splitting them into individual tracks.

    The version of the OpenAPI document: 1.0.0
    Generated by OpenAPI Generator (https://openapi-generator.tech)

    Do not edit the class manually.
"""  # noqa: E501


import unittest

from dj_set_downloader.models.server_cancel_response import ServerCancelResponse

class TestServerCancelResponse(unittest.TestCase):
    """ServerCancelResponse unit test stubs"""

    def setUp(self):
        pass

    def tearDown(self):
        pass

    def make_instance(self, include_optional) -> ServerCancelResponse:
        """Test ServerCancelResponse
            include_optional is a boolean, when False only required
            params are included, when True both required and
            optional params are included """
        # uncomment below to create an instance of `ServerCancelResponse`
        """
        model = ServerCancelResponse()
        if include_optional:
            return ServerCancelResponse(
                message = ''
            )
        else:
            return ServerCancelResponse(
        )
        """

    def testServerCancelResponse(self):
        """Test ServerCancelResponse"""
        # inst_req_only = self.make_instance(include_optional=False)
        # inst_req_and_optional = self.make_instance(include_optional=True)

if __name__ == '__main__':
    unittest.main()
