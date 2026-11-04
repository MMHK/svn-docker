import { t } from '../i18n';
import { ApiError } from './client';

export interface DescribedError {
  title: string;
  detail: string;
}

/** Normalises anything thrown by the API client into an Error. */
export function toError(cause: unknown): Error {
  if (cause instanceof Error) {
    return cause;
  }

  return new Error(typeof cause === 'string' ? cause : 'unknown error');
}

/**
 * Turns an API failure into a human readable title/detail pair. Called from
 * render code so the strings follow the active locale.
 */
export function describeError(error: Error): DescribedError {
  if (!(error instanceof ApiError)) {
    return { title: t('common.error'), detail: error.message };
  }

  switch (error.status) {
    case 0:
      return { title: t('common.networkError'), detail: t('common.networkErrorHint') };
    case 401:
      return { title: t('common.unauthorized'), detail: t('common.unauthorizedHint') };
    case 403:
      return { title: t('common.forbidden'), detail: t('common.forbiddenHint') };
    case 404:
      return { title: t('common.notFoundError'), detail: t('common.notFoundHint') };
    case 409:
      return { title: t('common.conflict'), detail: t('common.conflictHint') };
    case 422:
      // The API puts the offending field and reason in the message, so show it
      // verbatim instead of hiding it behind a generic "invalid request".
      return { title: t('common.validation'), detail: error.message };
    case 429:
      return { title: t('common.rateLimited'), detail: t('common.rateLimitedHint') };
    case 500:
      if (error.code === 'operation_failed') {
        return { title: t('common.operationFailed'), detail: error.message };
      }

      return {
        title: `${t('common.error')} ${error.status}`,
        detail: error.code === '' ? error.message : `${error.code}: ${error.message}`,
      };
    case 503:
      return { title: t('common.setupRequired'), detail: t('common.setupRequiredHint') };
    default:
      return {
        title: `${t('common.error')} ${error.status}`,
        detail: error.code === '' ? error.message : `${error.code}: ${error.message}`,
      };
  }
}
