/**
 * React Query hooks for push notifications
 */

import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import {
  getCurrentSubscription,
  updateNotificationPreferences,
  type NotificationPreferences,
  type UserSubscription,
} from '../api/push-notifications';
import {
  enablePushNotifications,
  disablePushNotifications,
  isPushNotificationSupported,
  getNotificationPermission,
} from '../push-notification-manager';

/**
 * Query key factory for push notifications
 */
export const pushNotificationKeys = {
  all: ['push-notifications'] as const,
  subscription: (projectName: string) =>
    [...pushNotificationKeys.all, 'subscription', projectName] as const,
  permission: ['push-permission'] as const,
};

/**
 * Hook to get current push notification subscription
 */
export function usePushSubscription(projectName: string) {
  return useQuery({
    queryKey: pushNotificationKeys.subscription(projectName),
    queryFn: () => getCurrentSubscription(projectName),
    enabled: isPushNotificationSupported(),
    staleTime: 5 * 60 * 1000,
  });
}

/**
 * Hook to get notification permission status
 */
export function useNotificationPermission() {
  return useQuery({
    queryKey: pushNotificationKeys.permission,
    queryFn: () => getNotificationPermission(),
    staleTime: 60 * 1000,
    refetchOnWindowFocus: true,
  });
}

/**
 * Hook to enable push notifications
 */
export function useEnablePushNotifications(projectName: string) {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (preferences: NotificationPreferences) =>
      enablePushNotifications(projectName, preferences),
    onSuccess: () => {
      queryClient.invalidateQueries({
        queryKey: pushNotificationKeys.subscription(projectName),
      });
      queryClient.invalidateQueries({
        queryKey: pushNotificationKeys.permission,
      });
    },
  });
}

/**
 * Hook to disable push notifications
 */
export function useDisablePushNotifications(projectName: string) {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: () => disablePushNotifications(projectName),
    onSuccess: () => {
      queryClient.invalidateQueries({
        queryKey: pushNotificationKeys.subscription(projectName),
      });
    },
  });
}

/**
 * Hook to update notification preferences
 */
export function useUpdateNotificationPreferences(projectName: string) {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: ({
      subscriptionId,
      preferences,
    }: {
      subscriptionId: string;
      preferences: NotificationPreferences;
    }) => updateNotificationPreferences(projectName, subscriptionId, preferences),
    onMutate: async ({ preferences }) => {
      // Cancel outgoing refetches
      await queryClient.cancelQueries({
        queryKey: pushNotificationKeys.subscription(projectName),
      });

      // Snapshot previous value
      const previousSubscription = queryClient.getQueryData<UserSubscription>(
        pushNotificationKeys.subscription(projectName)
      );

      // Optimistically update
      if (previousSubscription) {
        queryClient.setQueryData(pushNotificationKeys.subscription(projectName), {
          ...previousSubscription,
          preferences,
        });
      }

      return { previousSubscription };
    },
    onError: (_error, _variables, context) => {
      // Rollback on error
      if (context?.previousSubscription) {
        queryClient.setQueryData(
          pushNotificationKeys.subscription(projectName),
          context.previousSubscription
        );
      }
    },
    onSettled: () => {
      queryClient.invalidateQueries({
        queryKey: pushNotificationKeys.subscription(projectName),
      });
    },
  });
}
