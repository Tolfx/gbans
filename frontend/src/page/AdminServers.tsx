import React, { useEffect, useState } from 'react';
import NiceModal from '@ebay/nice-modal-react';
import CreateIcon from '@mui/icons-material/Create';
import DeleteIcon from '@mui/icons-material/Delete';
import EditIcon from '@mui/icons-material/Edit';
import StorageIcon from '@mui/icons-material/Storage';
import Button from '@mui/material/Button';
import ButtonGroup from '@mui/material/ButtonGroup';
import IconButton from '@mui/material/IconButton';
import Stack from '@mui/material/Stack';
import Tooltip from '@mui/material/Tooltip';
import Grid from '@mui/material/Unstable_Grid2';
import { apiGetServersAdmin, Server, ServerQueryFilter } from '../api';
import { ContainerWithHeader } from '../component/ContainerWithHeader';
import { Order, RowsPerPage } from '../component/DataTable';
import { LazyTable } from '../component/LazyTable';
import { TableCellBool } from '../component/TableCellBool';
import { ModalServerDelete, ModalServerEditor } from '../component/modal';
import { ServerEditorModal } from '../component/modal/ServerEditorModal';
import { logErr } from '../util/errors';

export const AdminServers = () => {
    const [bans, setBans] = useState<Server[]>([]);
    const [sortOrder, setSortOrder] = useState<Order>('desc');
    const [sortColumn, setSortColumn] = useState<keyof Server>('short_name');
    const [rowPerPageCount, setRowPerPageCount] = useState<number>(
        RowsPerPage.TwentyFive
    );
    const [page, setPage] = useState(0);
    const [totalRows, setTotalRows] = useState<number>(0);
    const [deleted] = useState(false);

    useEffect(() => {
        const abortController = new AbortController();
        const opts: ServerQueryFilter = {
            limit: rowPerPageCount,
            offset: page * rowPerPageCount,
            order_by: sortColumn,
            desc: sortOrder == 'desc',
            deleted: deleted,
            include_disabled: true
        };
        apiGetServersAdmin(opts, abortController)
            .then((resp) => {
                setBans(resp.data);
                setTotalRows(resp.count);
                if (page * rowPerPageCount > resp.count) {
                    setPage(0);
                }
            })
            .catch((e) => {
                logErr(e);
            });
    }, [deleted, page, rowPerPageCount, sortColumn, sortOrder]);
    return (
        <Grid container spacing={2}>
            <Grid xs={12}>
                <Stack spacing={2}>
                    <ButtonGroup>
                        <Button
                            variant={'contained'}
                            color={'secondary'}
                            startIcon={<CreateIcon />}
                            sx={{ marginRight: 2 }}
                            onClick={async () => {
                                await NiceModal.show(ServerEditorModal, {});
                            }}
                        >
                            Create Server
                        </Button>
                    </ButtonGroup>
                    <ContainerWithHeader
                        title={'Servers'}
                        iconLeft={<StorageIcon />}
                    >
                        <LazyTable<Server>
                            showPager={true}
                            count={totalRows}
                            rows={bans}
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
                                    tooltip: 'Name',
                                    label: 'Name',
                                    sortKey: 'short_name',
                                    align: 'left',
                                    sortable: true
                                },
                                {
                                    tooltip: 'Name Long',
                                    label: 'Name Long',
                                    sortKey: 'name',
                                    align: 'left',
                                    sortable: true
                                },
                                {
                                    tooltip: 'Address',
                                    label: 'Address',
                                    sortKey: 'address',
                                    align: 'left',
                                    sortable: true
                                },
                                {
                                    tooltip: 'Port',
                                    label: 'Port',
                                    sortKey: 'port',
                                    align: 'left',
                                    sortable: true
                                },
                                {
                                    tooltip: 'RCON Password',
                                    label: 'rcon',
                                    sortKey: 'rcon',
                                    align: 'left'
                                },
                                {
                                    tooltip: 'Region',
                                    label: 'Region',
                                    sortKey: 'region',
                                    align: 'left',
                                    sortable: true
                                },
                                {
                                    tooltip: 'CC',
                                    label: 'CC',
                                    sortKey: 'cc',
                                    align: 'left',
                                    sortable: true
                                },
                                {
                                    tooltip: 'Enabled',
                                    label: 'En.',
                                    sortKey: 'is_enabled',
                                    sortable: true,
                                    align: 'center',
                                    renderer: (row) => (
                                        <TableCellBool
                                            enabled={row.is_enabled}
                                        />
                                    )
                                },
                                {
                                    label: 'Act.',
                                    tooltip: 'Actions',
                                    sortable: false,
                                    align: 'center',
                                    renderer: (row) => (
                                        <ButtonGroup fullWidth>
                                            <IconButton
                                                color={'warning'}
                                                onClick={async () => {
                                                    await NiceModal.show(
                                                        ModalServerEditor,
                                                        {
                                                            server: row
                                                        }
                                                    );
                                                }}
                                            >
                                                <Tooltip title={'Edit Server'}>
                                                    <EditIcon />
                                                </Tooltip>
                                            </IconButton>
                                            <IconButton
                                                color={'warning'}
                                                onClick={async () => {
                                                    await NiceModal.show(
                                                        ModalServerDelete,
                                                        { server: row }
                                                    );
                                                }}
                                            >
                                                <Tooltip
                                                    title={'Delete Server'}
                                                >
                                                    <DeleteIcon
                                                        color={'error'}
                                                    />
                                                </Tooltip>
                                            </IconButton>
                                        </ButtonGroup>
                                    )
                                }
                            ]}
                        />
                    </ContainerWithHeader>
                </Stack>
            </Grid>
        </Grid>
    );
};
