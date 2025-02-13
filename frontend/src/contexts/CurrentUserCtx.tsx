import { createContext, useContext } from 'react';
import {
    defaultAvatarHash,
    PermissionLevel,
    userKey,
    UserProfile
} from '../api';

export const GuestProfile: UserProfile = {
    updated_on: new Date(),
    created_on: new Date(),
    permission_level: PermissionLevel.Guest,
    discord_id: '',
    avatarhash: defaultAvatarHash,
    steam_id: '',
    ban_id: 0,
    name: 'Guest',
    muted: false
};

export type CurrentUser = {
    currentUser: UserProfile;
    setCurrentUser: (profile: UserProfile) => void;
};

export const CurrentUserCtx = createContext<CurrentUser>({
    currentUser: GuestProfile,
    setCurrentUser: (profile: UserProfile) => {
        try {
            localStorage.setItem(userKey, JSON.stringify(profile));
        } catch (e) {
            return;
        }
    }
});

export const hasPermission = (
    profile: UserProfile,
    permission: PermissionLevel
): boolean => {
    return profile.permission_level >= permission;
};

export const useCurrentUserCtx = () => useContext(CurrentUserCtx);
