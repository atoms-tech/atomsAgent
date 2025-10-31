"""
Exception classes for ChatServer SDK
"""


class ChatServerError(Exception):
    """Base exception for all ChatServer errors"""
    
    def __init__(self, message: str, status_code: int = None, response = None):
        super().__init__(message)
        self.message = message
        self.status_code = status_code
        self.response = response


class BadRequestError(ChatServerError):
    """Raised for 400 Bad Request errors"""
    pass


class UnauthorizedError(ChatServerError):
    """Raised for 401 Unauthorized errors"""
    pass


class ForbiddenError(ChatServerError):
    """Raised for 403 Forbidden errors"""
    pass


class NotFoundError(ChatServerError):
    """Raised for 404 Not Found errors"""
    pass


class InternalServerError(ChatServerError):
    """Raised for 500 Internal Server Error"""
    pass


def _raise_for_status(response):
    """Raise appropriate exception based on HTTP status code"""
    if response.status_code >= 400:
        try:
            error_data = response.json()
            if 'error' in error_data:
                message = error_data['error'].get('message', f"HTTP {response.status_code}")
                error_type = error_data['error'].get('type', 'unknown')
            else:
                message = f"HTTP {response.status_code}"
                error_type = 'unknown'
        except:
            message = f"HTTP {response.status_code}"
            error_type = 'unknown'
        
        if response.status_code == 400:
            raise BadRequestError(message, response.status_code, response)
        elif response.status_code == 401:
            raise UnauthorizedError(message, response.status_code, response)
        elif response.status_code == 403:
            raise ForbiddenError(message, response.status_code, response)
        elif response.status_code == 404:
            raise NotFoundError(message, response.status_code, response)
        elif response.status_code >= 500:
            raise InternalServerError(message, response.status_code, response)
        else:
            raise ChatServerError(message, response.status_code, response)
