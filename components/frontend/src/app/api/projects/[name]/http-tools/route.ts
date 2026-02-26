import { createProxyRouteHandlers } from '@/lib/api-route-helpers';

export const { GET, PUT } = createProxyRouteHandlers(
  (name) => `/projects/${encodeURIComponent(name)}/http-tools`
);
