/**
 * Error classes for ChatServer SDK
 */

import type { ChatServerError } from './types';

export class ChatServerException extends Error implements ChatServerError {
  public status?: number;
  public response?: Response;

  constructor(message: string, status?: number, response?: Response) {
    super(message);
    this.name = 'ChatServerException';
    if (status !== undefined) {
      this.status = status;
    }
    if (response !== undefined) {
      this.response = response;
    }
  }
}

export class BadRequestError extends ChatServerException {
  constructor(message: string, response?: Response) {
    super(message, 400, response);
    this.name = 'BadRequestError';
  }
}

export class UnauthorizedError extends ChatServerException {
  constructor(message: string, response?: Response) {
    super(message, 401, response);
    this.name = 'UnauthorizedError';
  }
}

export class ForbiddenError extends ChatServerException {
  constructor(message: string, response?: Response) {
    super(message, 403, response);
    this.name = 'ForbiddenError';
  }
}

export class NotFoundError extends ChatServerException {
  constructor(message: string, response?: Response) {
    super(message, 404, response);
    this.name = 'NotFoundError';
  }
}

export class InternalServerError extends ChatServerException {
  constructor(message: string, response?: Response) {
    super(message, 500, response);
    this.name = 'InternalServerError';
  }
}

export function throwForStatus(response: Response): void {
  if (response.status >= 400) {
    let message = `HTTP ${response.status}`;
    
    response.clone().json().then((data: any) => {
      if (data.error && data.error.message) {
        message = data.error.message;
      }
    }).catch(() => {
      // Ignore JSON parsing errors
    });

    switch (response.status) {
      case 400:
        throw new BadRequestError(message, response);
      case 401:
        throw new UnauthorizedError(message, response);
      case 403:
        throw new ForbiddenError(message, response);
      case 404:
        throw new NotFoundError(message, response);
      default:
        if (response.status >= 500) {
          throw new InternalServerError(message, response);
        } else {
          throw new ChatServerException(message, response.status, response);
        }
    }
  }
}
