import React, { useMemo } from 'react';
import Stack from '@mui/material/Stack';
import { apiCreateBanSteam, BanReason, BanType, Duration } from '../api';
import { Heading } from './Heading';
import * as yup from 'yup';
import { useFormik } from 'formik';
import GavelIcon from '@mui/icons-material/Gavel';
import { Dialog, DialogContent, DialogTitle } from '@mui/material';
import {
    SteamIdField,
    SteamIDInputValue,
    steamIdValidator
} from './formik/SteamIdField';
import { logErr } from '../util/errors';
import { useUserFlashCtx } from '../contexts/UserFlashCtx';
import { BanTypeField, BanTypeFieldValidator } from './formik/BanTypeField';
import {
    BanReasonField,
    BanReasonFieldValidator
} from './formik/BanReasonField';
import { DurationField, DurationFieldValidator } from './formik/DurationField';
import {
    DurationCustomField,
    DurationCustomFieldValidator
} from './formik/DurationCustomField';
import { NoteField, NoteFieldValidator } from './formik/NoteField';
import { ReportIdField, ReportIdFieldValidator } from './formik/ReportIdField';
import { ModalButtons } from './formik/ModalButtons';
import {
    BanReasonTextField,
    BanReasonTextFieldValidator
} from './formik/BanReasonTextField';
import { IncludeFriendsField } from './formik/IncludeFriendsField';

export interface BanModalProps {
    open: boolean;
    setOpen: (open: boolean) => void;
    reportId?: number;
    steamId?: string;
}

interface BanSteamFormValues extends SteamIDInputValue {
    reportId?: number;
    banType: BanType;
    reason: BanReason;
    reasonText: string;
    duration: Duration;
    durationCustom: string;
    note: string;
    include_friends: boolean;
}

const validationSchema = yup.object({
    steam_id: steamIdValidator,
    reportId: ReportIdFieldValidator,
    banType: BanTypeFieldValidator,
    reason: BanReasonFieldValidator,
    reasonText: BanReasonTextFieldValidator,
    duration: DurationFieldValidator,
    durationCustom: DurationCustomFieldValidator,
    note: NoteFieldValidator
});

export const BanSteamModal = ({
    open,
    setOpen,
    steamId,
    reportId
}: BanModalProps) => {
    const { sendFlash } = useUserFlashCtx();

    const isReadOnlySid = useMemo(() => {
        return !!steamId;
    }, [steamId]);

    const formik = useFormik<BanSteamFormValues>({
        initialValues: {
            banType: BanType.NoComm,
            duration: Duration.dur2w,
            durationCustom: '',
            note: '',
            reason: BanReason.Cheating,
            steam_id: steamId ?? '',
            reasonText: '',
            reportId: reportId,
            include_friends: false
        },
        validateOnBlur: true,
        validateOnChange: false,
        onReset: () => {
            alert('reset!');
        },
        validationSchema: validationSchema,
        onSubmit: async (values) => {
            try {
                await apiCreateBanSteam({
                    note: values.note,
                    ban_type: values.banType,
                    duration: values.duration,
                    reason: values.reason,
                    reason_text: values.reasonText,
                    report_id: values.reportId,
                    target_id: values.steam_id,
                    include_friends: values.include_friends
                });
                sendFlash('success', 'Ban created successfully');
            } catch (e) {
                logErr(e);
                sendFlash('error', 'Error saving ban');
            } finally {
                setOpen(false);
            }
        }
    });

    const formId = 'banSteamForm';

    return (
        <form onSubmit={formik.handleSubmit} id={formId}>
            <Dialog
                fullWidth
                open={open}
                onClose={() => {
                    setOpen(false);
                }}
            >
                <DialogTitle component={Heading} iconLeft={<GavelIcon />}>
                    Ban Steam Profile
                </DialogTitle>

                <DialogContent>
                    <Stack spacing={2}>
                        <SteamIdField
                            formik={formik}
                            fullWidth
                            isReadOnly={isReadOnlySid}
                        />
                        <ReportIdField formik={formik} />
                        <BanTypeField formik={formik} />
                        <BanReasonField formik={formik} />
                        <BanReasonTextField formik={formik} />
                        <IncludeFriendsField formik={formik} />
                        <DurationField formik={formik} />
                        <DurationCustomField formik={formik} />
                        <NoteField formik={formik} />
                    </Stack>
                </DialogContent>
                <ModalButtons
                    formId={formId}
                    setOpen={setOpen}
                    inProgress={false}
                />
            </Dialog>
        </form>
    );
};
