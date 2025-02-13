import React, { useState } from 'react';
import NiceModal from '@ebay/nice-modal-react';
import AddBoxIcon from '@mui/icons-material/AddBox';
import DeleteForeverIcon from '@mui/icons-material/DeleteForever';
import EditIcon from '@mui/icons-material/Edit';
import FilterAltIcon from '@mui/icons-material/FilterAlt';
import Button from '@mui/material/Button';
import ButtonGroup from '@mui/material/ButtonGroup';
import IconButton from '@mui/material/IconButton';
import Tooltip from '@mui/material/Tooltip';
import Grid from '@mui/material/Unstable_Grid2';
import { Filter, FilterAction, filterActionString } from '../api/filters';
import { ContainerWithHeader } from '../component/ContainerWithHeader';
import { ModalFilterDelete, ModalFilterEditor } from '../component/modal';
import { LazyTable, Order, RowsPerPage } from '../component/table/LazyTable';
import { useUserFlashCtx } from '../contexts/UserFlashCtx';
import { useWordFilters } from '../hooks/useWordFilters';

export const AdminFiltersPage = () => {
    const [sortOrder, setSortOrder] = useState<Order>('desc');
    const [sortColumn, setSortColumn] = useState<keyof Filter>('filter_id');
    const [rowPerPageCount, setRowPerPageCount] = useState<number>(
        RowsPerPage.TwentyFive
    );
    const [page, setPage] = useState(0);
    const { sendFlash } = useUserFlashCtx();

    const { data, count } = useWordFilters({
        order_by: sortColumn,
        desc: sortOrder == 'desc',
        limit: rowPerPageCount,
        offset: page * rowPerPageCount
    });

    return (
        <Grid container spacing={2}>
            <Grid xs={12}>
                <ButtonGroup
                    variant="contained"
                    aria-label="outlined primary button group"
                >
                    <Button
                        startIcon={<AddBoxIcon />}
                        color={'success'}
                        onClick={async () => {
                            try {
                                const resp = await NiceModal.show<Filter>(
                                    ModalFilterEditor,
                                    {
                                        defaultPattern: '',
                                        defaultIsRegex: false
                                    }
                                );
                                sendFlash(
                                    'success',
                                    `Filter created successfully: ${resp.filter_id}`
                                );
                            } catch (e) {
                                sendFlash('error', `${e}`);
                            }
                        }}
                    >
                        New
                    </Button>
                </ButtonGroup>
            </Grid>
            <Grid xs={12}>
                <ContainerWithHeader
                    title={'Word Filters'}
                    iconLeft={<FilterAltIcon />}
                >
                    <LazyTable<Filter>
                        showPager={true}
                        count={count}
                        rows={data}
                        page={page}
                        rowsPerPage={rowPerPageCount}
                        sortOrder={sortOrder}
                        sortColumn={sortColumn}
                        onSortColumnChanged={async (column) => {
                            setSortColumn(column);
                        }}
                        onSortOrderChanged={async (direction) => {
                            setSortOrder(direction);
                        }}
                        onPageChange={(_, newPage: number) => {
                            setPage(newPage);
                        }}
                        onRowsPerPageChange={(
                            event: React.ChangeEvent<
                                HTMLInputElement | HTMLTextAreaElement
                            >
                        ) => {
                            setRowPerPageCount(
                                parseInt(event.target.value, 10)
                            );
                            setPage(0);
                        }}
                        columns={[
                            {
                                label: 'Pattern',
                                tooltip: 'Pattern',
                                sortKey: 'pattern',
                                sortable: true,
                                align: 'left',
                                renderer: (row) => {
                                    return row.pattern as string;
                                }
                            },
                            {
                                label: 'Regex',
                                tooltip: 'Regular Expression',
                                sortKey: 'is_regex',
                                sortable: false,
                                align: 'right',
                                renderer: (row) => {
                                    return row.is_regex ? 'true' : 'false';
                                }
                            },
                            {
                                label: 'Action',
                                tooltip: 'What will happen when its triggered',
                                sortKey: 'action',
                                sortable: true,
                                align: 'right',
                                renderer: (row) => {
                                    return filterActionString(row.action);
                                }
                            },
                            {
                                label: 'Duration',
                                tooltip:
                                    'Duration when the action is triggered',
                                sortKey: 'duration',
                                sortable: false,
                                align: 'right',
                                renderer: (row) => {
                                    return row.action == FilterAction.Kick
                                        ? ''
                                        : row.duration;
                                }
                            },
                            {
                                label: 'Triggered',
                                tooltip:
                                    'Number of times the filter has been triggered',
                                sortKey: 'trigger_count',
                                sortable: true,
                                sortType: 'number',
                                align: 'right',
                                renderer: (row) => {
                                    return row.trigger_count;
                                }
                            },
                            {
                                label: 'Actions',
                                tooltip: 'Action',
                                virtualKey: 'actions',
                                sortable: false,
                                align: 'right',
                                virtual: true,
                                renderer: (row) => {
                                    return (
                                        <ButtonGroup>
                                            <Tooltip title={'Edit filter'}>
                                                <IconButton
                                                    color={'warning'}
                                                    onClick={async () => {
                                                        await NiceModal.show(
                                                            ModalFilterEditor,
                                                            {
                                                                filter: row
                                                            }
                                                        );
                                                    }}
                                                >
                                                    <EditIcon />
                                                </IconButton>
                                            </Tooltip>
                                            <Tooltip title={'Delete filter'}>
                                                <IconButton
                                                    color={'error'}
                                                    onClick={async () => {
                                                        await NiceModal.show(
                                                            ModalFilterDelete,
                                                            {
                                                                record: row
                                                            }
                                                        );
                                                    }}
                                                >
                                                    <DeleteForeverIcon />
                                                </IconButton>
                                            </Tooltip>
                                        </ButtonGroup>
                                    );
                                }
                            }
                        ]}
                    />
                </ContainerWithHeader>
            </Grid>
        </Grid>
    );
};
