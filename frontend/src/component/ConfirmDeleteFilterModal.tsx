import Stack from '@mui/material/Stack';
import React, { useCallback } from 'react';
import { apiDeleteFilter, Filter } from '../api/filters';
import { useUserFlashCtx } from '../contexts/UserFlashCtx';
import { logErr } from '../util/errors';
import { ConfirmationModal, ConfirmationModalProps } from './ConfirmationModal';
import { Heading } from './Heading';

export interface ConfirmDeleteFilterModalProps
    extends ConfirmationModalProps<Filter> {
    record: Filter;
}

export const ConfirmDeleteFilterModal = ({
    open,
    setOpen,
    onSuccess,
    record
}: ConfirmDeleteFilterModalProps) => {
    const { sendFlash } = useUserFlashCtx();

    const handleSubmit = useCallback(() => {
        if (!record.filter_id) {
            logErr(new Error('filter_id not present, cannot delete'));
            return;
        }
        apiDeleteFilter(record.filter_id)
            .then(() => {
                sendFlash('success', `Deleted filter successfully`);
                onSuccess && onSuccess(record);
            })
            .catch((err) => {
                sendFlash('error', `Failed to delete filter: ${err}`);
            });
    }, [record, sendFlash, onSuccess]);

    return (
        <ConfirmationModal
            open={open}
            setOpen={setOpen}
            onSuccess={() => {
                setOpen(false);
            }}
            onCancel={() => {
                setOpen(false);
            }}
            onAccept={() => {
                handleSubmit();
            }}
            aria-labelledby="modal-title"
            aria-describedby="modal-description"
        >
            <Stack spacing={2}>
                <Heading>{`Delete word filter (#${record.filter_id})?`}</Heading>
            </Stack>
        </ConfirmationModal>
    );
};
