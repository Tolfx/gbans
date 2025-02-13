import { useEffect, useState } from 'react';
import { LazyResult } from '../component/table/LazyTableSimple';
import { logErr } from '../util/errors';
import {
    apiCall,
    APIError,
    DateRange,
    PermissionLevel,
    TimeStamped,
    transformDateRange,
    transformTimeStampedDates
} from './common';
import { EmptyUUID } from './const';
import { Asset } from './media';

export interface Contest extends DateRange, TimeStamped {
    contest_id: string;
    title: string;
    description: string;
    public: boolean;
    hide_submissions: boolean;
    max_submissions: number;
    media_types: string;
    deleted: boolean;
    voting: boolean;
    min_permission_level: PermissionLevel;
    down_votes: boolean;
    num_entries: number;
}

export const apiContestSave = async (contest: Contest) =>
    contest.contest_id == EmptyUUID
        ? await apiCall<Contest, Contest>(`/api/contests`, 'POST', contest)
        : await apiCall<Contest, Contest>(
              `/api/contests/${contest.contest_id}`,
              'PUT',
              contest
          );

export const apiContests = async () => {
    const resp = await apiCall<LazyResult<Contest>>(`/api/contests`, 'GET');
    if (resp.data) {
        resp.data = resp.data
            .map(transformDateRange)
            .map(transformTimeStampedDates);
    }

    return resp;
};

export const apiContest = async (contest_id: string) => {
    const contest = await apiCall<Contest>(
        `/api/contests/${contest_id}`,
        'GET'
    );
    return transformDateRange(contest);
};

export const apiContestEntries = async (contest_id: string) => {
    try {
        const entries = await apiCall<ContestEntry[]>(
            `/api/contests/${contest_id}/entries`,
            'GET'
        );
        return entries.map(transformTimeStampedDates);
    } catch (e) {
        logErr(e);
        return [];
    }
};
export const apiContestDelete = async (contest_id: string) =>
    await apiCall<Contest>(`/api/contests/${contest_id}`, 'DELETE');

export interface ContestEntry extends TimeStamped {
    contest_id: string;
    contest_entry_id: string;
    description: string;
    asset_id: string;
    steam_id: string;
    placement: number;
    personaname: string;
    avatar_hash: string;
    votes_up: number;
    votes_down: number;
    asset: Asset;
}

export const apiContestEntryDelete = async (contest_entry_id: string) =>
    await apiCall(`/api/contest_entry/${contest_entry_id}`, 'DELETE');

export const apiContestEntrySave = async (
    contest_id: string,
    description: string,
    asset_id: string
) =>
    await apiCall<ContestEntry>(`/api/contests/${contest_id}/submit`, 'POST', {
        description,
        asset_id
    });

interface VoteResult {
    current_vote: string;
}

export const apiContestEntryVote = async (
    contest_id: string,
    contest_entry_id: string,
    upvote: boolean
) =>
    await apiCall<VoteResult>(
        `/api/contests/${contest_id}/vote/${contest_entry_id}/${
            upvote ? 'up' : 'down'
        }`,
        'GET'
    );

export const useContests = () => {
    const [loading, setLoading] = useState(false);
    const [contests, setContests] = useState<Contest[]>([]);

    useEffect(() => {
        apiContests()
            .then((contests) => {
                setContests(contests.data);
            })
            .catch((e) => {
                setContests([]);
                logErr(e);
            })
            .finally(() => {
                setLoading(false);
            });
    }, []);

    return { loading, contests };
};

export const useContest = (contest_id?: string) => {
    const [loading, setLoading] = useState(false);
    const [contest, setContest] = useState<Contest>();
    const [error, setError] = useState<APIError>();

    useEffect(() => {
        if (!contest_id) {
            return;
        }

        apiContest(contest_id)
            .then((contest) => {
                setContest(contest);
            })
            .catch((reason) => {
                setError(reason as APIError);
            })
            .finally(() => {
                setLoading(false);
            });
    }, [contest_id]);

    return { loading, contest, error };
};
