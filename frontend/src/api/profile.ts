import { LazyResult } from '../component/table/LazyTableSimple';
import { parseDateTime } from '../util/text';
import {
    apiCall,
    PermissionLevel,
    QueryFilter,
    TimeStamped,
    transformCreatedOnDate,
    transformTimeStampedDates
} from './common';

export const defaultAvatarHash = 'fef49e7fa7e1997310d705b2a6158ff8dc1cdfeb';

export enum profileState {
    Incomplete = 0,
    Setup = 1
}

export enum communityVisibilityState {
    Private = 1,
    FriendOnly = 2,
    Public = 3
}

enum NotificationSeverity {
    SeverityInfo,
    SeverityWarn,
    SeverityError
}

export interface UserNotification {
    person_notification_id: number;
    steam_id: string;
    read: boolean;
    deleted: boolean;
    severity: NotificationSeverity;
    message: string;
    link: string;
    count: number;
    created_on: string;
}

export interface UserProfile extends TimeStamped {
    steam_id: string;
    permission_level: PermissionLevel;
    discord_id: string;
    name: string;
    avatarhash: string;
    ban_id: number;
    muted: boolean;
}

export interface Person extends UserProfile {
    // PlayerSummaries shape
    steamid: string;
    communityvisibilitystate: communityVisibilityState;
    profilestate: profileState;
    personaname: string;
    profileurl: string;
    avatarmedium: string;
    avatarhash: string;
    personastate: number;
    realname: string;
    primaryclanid: string; // ? should be number
    timecreated: number;
    personastateflags: number;
    loccountrycode: string;
    locstatecode: string;
    loccityid: number;

    // BanStates
    community_banned: boolean;
    vac_bans: number;
    game_bans: number;
    economy_ban: string;
    days_since_last_ban: number;
    updated_on_steam: Date;
    ip_addr: string;
}

export interface PlayerProfile {
    player: Person;
    friends?: Person[];
    settings: PersonSettings;
}

export const apiGetProfile = async (
    query: string,
    abortController?: AbortController
) =>
    await apiCall<PlayerProfile>(
        `/api/profile?query=${query}`,
        'GET',
        undefined,
        abortController
    );

export const apiGetCurrentProfile = async (abortController: AbortController) =>
    await apiCall<UserProfile>(
        `/api/current_profile`,
        'GET',
        undefined,
        abortController
    );

export interface PlayerQuery extends QueryFilter<Person> {
    steam_id: string;
    personaname: string;
    ip: string;
}

export const apiSearchPeople = async (
    opts: PlayerQuery,
    abortController?: AbortController
) => {
    const people = await apiCall<LazyResult<Person>>(
        `/api/players`,
        'POST',
        opts,
        abortController
    );
    people.data = people.data.map(transformTimeStampedDates);
    return people;
};

export const apiLinkDiscord = async (opts: { code: string }) =>
    await apiCall(`/api/auth/discord?code=${opts.code}`, 'GET');

export interface PersonIPRecord {
    ip_addr: string;
    created_on: Date;
    city_name: string;
    country_name: string;
    country_code: string;
    as_name: string;
    as_num: number;
    isp: string;
    usage_type: string;
    threat: string;
    domain: string;
}

export interface PersonConnection {
    person_connection_id: bigint;
    ip_addr: string;
    steam_id: string;
    persona_name: string;
    created_on: Date;
    ip_info: PersonIPRecord;
    server_id?: number;
    server_name_short?: string;
    server_name?: string;
}

export interface PersonMessage {
    person_message_id: number;
    steam_id: string;
    persona_name: string;
    server_name: string;
    server_id: number;
    body: string;
    team: boolean;
    created_on: Date;
    auto_filter_flagged: number;
    avatar_hash: string;
    pattern: string;
}

export const apiGetMessageContext = async (
    messageId: number,
    padding: number = 5
) => {
    const resp = await apiCall<PersonMessage[]>(
        `/api/message/${messageId}/context/${padding}`,
        'GET'
    );
    return resp.map((msg) => {
        return {
            ...msg,
            created_on: parseDateTime(msg.created_on as unknown as string)
        };
    });
};

export interface MessageQuery extends QueryFilter<PersonMessage> {
    personaname?: string;
    source_id?: string;
    query?: string;
    server_id?: number;
    date_start?: Date;
    date_end?: Date;
}

export const apiGetMessages = async (
    opts: MessageQuery,
    abortController?: AbortController
) => {
    const resp = await apiCall<LazyResult<PersonMessage>, MessageQuery>(
        `/api/messages`,
        'POST',
        opts,
        abortController
    );

    resp.data = resp.data.map((msg) => {
        return {
            ...msg,
            created_on: parseDateTime(msg.created_on as unknown as string)
        };
    });

    return resp;
};

export type NotificationsQuery = QueryFilter<UserNotification>;

export const apiGetNotifications = async (
    opts: NotificationsQuery,
    abortController: AbortController
) => {
    return await apiCall<LazyResult<UserNotification>>(
        `/api/current_profile/notifications`,
        'POST',
        opts,
        abortController
    );
};

export interface PersonConnectionQuery extends QueryFilter<PersonConnection> {
    cidr?: string;
    source_id?: string;
    server_id?: number;
    asn?: number;
}

export const apiGetConnections = async (
    opts: PersonConnectionQuery,
    abortController: AbortController
) => {
    const resp = await apiCall<
        LazyResult<PersonConnection>,
        PersonConnectionQuery
    >(`/api/connections`, 'POST', opts, abortController);

    resp.data = resp.data.map(transformCreatedOnDate);
    return resp;
};

interface PermissionUpdate {
    permission_level: PermissionLevel;
}

export const apiUpdatePlayerPermission = async (
    steamId: string,
    args: PermissionUpdate,
    abortController: AbortController
) =>
    transformTimeStampedDates(
        await apiCall<Person, PermissionUpdate>(
            `/api/player/${steamId}/permissions`,
            'PUT',
            args,
            abortController
        )
    );

export interface PersonSettings extends TimeStamped {
    person_settings_id: number;
    steam_id: string;
    forum_signature: string;
    forum_profile_messages: boolean;
    stats_hidden: boolean;
}

export const apiGetPersonSettings = async (
    abortController: AbortController
) => {
    return transformTimeStampedDates(
        await apiCall<PersonSettings>(
            `/api/current_profile/settings`,
            'GET',
            undefined,
            abortController
        )
    );
};

export const apSavePersonSettings = async (
    forum_signature: string,
    forum_profile_messages: boolean,
    stats_hidden: boolean,
    abortController?: AbortController
) => {
    return transformTimeStampedDates(
        await apiCall<PersonSettings>(
            `/api/current_profile/settings`,
            'POST',
            { forum_signature, forum_profile_messages, stats_hidden },
            abortController
        )
    );
};
