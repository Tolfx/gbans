import React, { useCallback, useState } from 'react';
import NiceModal, { muiDialogV5, useModal } from '@ebay/nice-modal-react';
import GroupsIcon from '@mui/icons-material/Groups';
import {
    Dialog,
    DialogActions,
    DialogContent,
    DialogTitle
} from '@mui/material';
import Stack from '@mui/material/Stack';
import { Formik } from 'formik';
import * as yup from 'yup';
import {
    apiCreateBanGroup,
    APIError,
    apiUpdateBanGroup,
    Duration,
    GroupBanRecord
} from '../../api';
import { Heading } from '../Heading';
import {
    DurationCustomField,
    DurationCustomFieldValidator
} from '../formik/DurationCustomField';
import { DurationField, DurationFieldValidator } from '../formik/DurationField';
import { ErrorField } from '../formik/ErrorField';
import { GroupIdField, groupIdFieldValidator } from '../formik/GroupIdField';
import { NoteField, NoteFieldValidator } from '../formik/NoteField';
import { SteamIdField, steamIdValidator } from '../formik/SteamIdField';
import { CancelButton, ResetButton, SubmitButton } from './Buttons';

export interface BanGroupFormValues {
    ban_group_id?: number;
    steam_id: string;
    group_id: string;
    duration: Duration;
    duration_custom: Date;
    note: string;
}

const validationSchema = yup.object({
    steam_id: steamIdValidator,
    group_id: groupIdFieldValidator,
    duration: DurationFieldValidator,
    duration_custom: DurationCustomFieldValidator,
    note: NoteFieldValidator
});

export interface BanGroupModalProps {
    existing?: GroupBanRecord;
}

export const BanGroupModal = NiceModal.create(
    ({ existing }: BanGroupModalProps) => {
        const [error, setError] = useState<string>();
        const modal = useModal();
        const onSubmit = useCallback(
            async (values: BanGroupFormValues) => {
                try {
                    if (existing != undefined && existing.ban_group_id > 0) {
                        modal.resolve(
                            await apiUpdateBanGroup(existing.ban_group_id, {
                                note: values.note,
                                valid_until: values.duration_custom,
                                target_id: values.steam_id
                            })
                        );
                    } else {
                        modal.resolve(
                            await apiCreateBanGroup({
                                group_id: values.group_id,
                                note: values.note,
                                duration: values.duration,
                                valid_until: values.duration_custom,
                                target_id: values.steam_id
                            })
                        );
                    }
                    await modal.hide();
                    setError(undefined);
                } catch (e) {
                    modal.reject(e);
                    if (e instanceof APIError) {
                        setError(e.message);
                    } else {
                        setError('Unknown internal error');
                    }
                }
            },
            [existing, modal]
        );

        return (
            <Formik
                onSubmit={onSubmit}
                id={'banGroupForm'}
                initialValues={{
                    ban_group_id: existing?.ban_group_id,
                    steam_id: existing ? existing.target_id : '',
                    duration: existing ? Duration.durCustom : Duration.dur2w,
                    duration_custom: existing
                        ? existing.valid_until
                        : new Date(),
                    note: existing ? existing.note : '',
                    group_id: existing ? existing.group_id : ''
                }}
                validateOnBlur={true}
                validateOnChange={false}
                validationSchema={validationSchema}
            >
                <Dialog fullWidth {...muiDialogV5(modal)}>
                    <DialogTitle component={Heading} iconLeft={<GroupsIcon />}>
                        Ban Steam Group
                    </DialogTitle>

                    <DialogContent>
                        <Stack spacing={2}>
                            <SteamIdField />
                            <GroupIdField />
                            <DurationField />
                            <DurationCustomField />
                            <NoteField />
                            <ErrorField error={error} />
                        </Stack>
                    </DialogContent>
                    <DialogActions>
                        <CancelButton />
                        <ResetButton />
                        <SubmitButton />
                    </DialogActions>
                </Dialog>
            </Formik>
        );
    }
);
