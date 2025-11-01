import axios from 'axios';

const resolveApiBaseUrl = () => {
  const envBaseUrl = (import.meta.env.VITE_API_BASE_URL as string | undefined) || '';
  const trimmedEnvBaseUrl = envBaseUrl.trim();
  if (trimmedEnvBaseUrl.length > 0) {
    return trimmedEnvBaseUrl;
  }

  if (import.meta.env.DEV) {
    return '/api';
  }

  if (typeof window === 'undefined') {
    return '/api';
  }

  const protocol = window.location.protocol;
  const hostname = window.location.hostname;
  const envPort = (import.meta.env.VITE_API_PORT as string | undefined) || '';
  const trimmedPort = envPort.trim();
  const targetPort = trimmedPort.length > 0 ? trimmedPort : '8080';

  if (targetPort === '80' || targetPort === '443' || targetPort === '') {
    return `${protocol}//${hostname}/api`;
  }

  return `${protocol}//${hostname}:${targetPort}/api`;
};

const api = axios.create({
  baseURL: resolveApiBaseUrl(),
  withCredentials: true,
  headers: {
    'Content-Type': 'application/json',
  },
});

// Request interceptor
api.interceptors.request.use(
  (config) => {
    return config;
  },
  (error) => {
    return Promise.reject(error);
  }
);

// Response interceptor
api.interceptors.response.use(
  (response) => response,
  (error) => {
    if (error.response?.status === 401) {
      // Redirect to login
      window.location.href = '/login';
    }
    return Promise.reject(error);
  }
);

export default api;

