from litestar.exceptions import ClientException
from litestar.status_codes import HTTP_400_BAD_REQUEST


class BadRequestException(ClientException):
    """Server knows the request method, but the target resource doesn't support this method."""

    status_code = HTTP_400_BAD_REQUEST
