import React, { useCallback, useState } from 'react';
import HelpIcon from '@mui/icons-material/Help';
import LeakAddIcon from '@mui/icons-material/LeakAdd';
import PersonSearchIcon from '@mui/icons-material/PersonSearch';
import VpnLockIcon from '@mui/icons-material/VpnLock';
import Box from '@mui/material/Box';
import ButtonGroup from '@mui/material/ButtonGroup';
import List from '@mui/material/List';
import ListItem from '@mui/material/ListItem';
import ListItemText from '@mui/material/ListItemText';
import Stack from '@mui/material/Stack';
import Tab from '@mui/material/Tab';
import Tabs from '@mui/material/Tabs';
import TextField from '@mui/material/TextField';
import Typography from '@mui/material/Typography';
import Grid from '@mui/material/Unstable_Grid2';
import { fromUnixTime } from 'date-fns';
import { Formik } from 'formik';
import IPCIDR from 'ip-cidr';
import * as yup from 'yup';
import { apiPlayersAtIP, Person } from '../api';
import { ContainerWithHeader } from '../component/ContainerWithHeader';
import {
    ipFieldValidator,
    NetworkBlockChecker
} from '../component/NetworkBlockChecker';
import { NetworkBlockSources } from '../component/NetworkBlockSources';
import { PersonCell } from '../component/PersonCell';
import { TabPanel } from '../component/TabPanel';
import { VCenterBox } from '../component/VCenterBox';
import { IPField } from '../component/formik/IPField';
import { SubmitButton } from '../component/modal/Buttons';
import { LazyTable, Order, RowsPerPage } from '../component/table/LazyTable';
import { logErr } from '../util/errors';
import { isValidSteamDate, renderDate, renderDateTime } from '../util/text';

interface NetworkInputProps {
    onValidChange: (cidr: string) => void;
}

export const NetworkInput = ({ onValidChange }: NetworkInputProps) => {
    const defaultHelperText = 'Enter a IP address or CIDR range';
    const [error, setError] = React.useState('');
    const [value, setValue] = React.useState('');
    const [helper, setHelper] = React.useState(defaultHelperText);

    const onChange = React.useCallback(
        (evt: React.ChangeEvent<HTMLInputElement>) => {
            const address = evt.target.value;
            if (address == '') {
                setError('');
                setValue(address);
                setHelper(defaultHelperText);
                return;
            }
            if (!address.match(`^([0-9./]+?)$`)) {
                return;
            }

            setValue(address);

            if (address.length > 0 && !IPCIDR.isValidAddress(address)) {
                setError('Invalid address');
                return;
            }

            setError('');

            try {
                const cidr = new IPCIDR(address);
                setHelper(`Total hosts in range: ${cidr.size}`);
                onValidChange(address);
            } catch (e) {
                if (IPCIDR.isValidAddress(address)) {
                    setHelper(`Total hosts in range: 1`);
                    onValidChange(address);
                }
                return;
            }
        },
        [onValidChange]
    );

    return (
        <TextField
            fullWidth
            error={Boolean(error.length)}
            id="outlined-error-helper-text"
            label="IP/CIDR"
            value={value}
            onChange={onChange}
            helperText={helper}
        />
    );
};

const validationSchema = yup.object({ ip: ipFieldValidator });

interface FindPlayerIPValues {
    ip: string;
}

const FindPlayerIP = () => {
    const [sortOrder, setSortOrder] = useState<Order>('desc');
    const [sortColumn, setSortColumn] = useState<keyof Person>('steam_id');
    const [page, setPage] = useState(0);
    const [rowPerPageCount, setRowPerPageCount] = useState<number>(
        RowsPerPage.TwentyFive
    );
    const [players, setPlayers] = useState<Person[]>([]);

    const onSubmit = useCallback(async (values: FindPlayerIPValues) => {
        try {
            const found = await apiPlayersAtIP(values.ip);
            setPlayers(found ?? []);
        } catch (e) {
            logErr(e);
        }
    }, []);

    return (
        <Formik<FindPlayerIPValues>
            onSubmit={onSubmit}
            initialValues={{ ip: '' }}
            validationSchema={validationSchema}
        >
            <Grid container padding={1} spacing={1}>
                <Grid xs={9}>
                    <Stack>
                        <IPField />
                    </Stack>
                </Grid>
                <Grid xs={3}>
                    <VCenterBox>
                        <ButtonGroup fullWidth>
                            <SubmitButton
                                label={'Find Players'}
                                startIcon={<PersonSearchIcon />}
                            />
                        </ButtonGroup>
                    </VCenterBox>
                </Grid>
                <Grid xs={12}>
                    <LazyTable
                        columns={[
                            {
                                label: 'Profile',
                                align: 'left',
                                sortable: true,
                                sortKey: 'steam_id',
                                tooltip: 'Profile at IP Address',
                                renderer: (obj) => {
                                    return (
                                        <PersonCell
                                            avatar_hash={obj.avatarhash}
                                            personaname={obj.personaname}
                                            steam_id={obj.steam_id}
                                        />
                                    );
                                }
                            },
                            {
                                label: 'Vac Bans',
                                tooltip: 'Amount of vac bans',
                                sortKey: 'vac_bans',
                                align: 'left',
                                sortable: true,
                                renderer: (row) => (
                                    <Typography variant={'body1'}>
                                        {row.vac_bans}
                                    </Typography>
                                )
                            },
                            {
                                label: 'Account Created',
                                tooltip: 'When the account was created',
                                sortKey: 'timecreated',
                                align: 'left',
                                sortable: true,
                                renderer: (row) => (
                                    <Typography variant={'body1'}>
                                        {!isValidSteamDate(
                                            fromUnixTime(row.timecreated)
                                        )
                                            ? 'Unknown'
                                            : renderDate(
                                                  fromUnixTime(row.timecreated)
                                              )}
                                    </Typography>
                                )
                            },
                            {
                                label: 'First Seen',
                                align: 'left',
                                sortable: true,
                                sortKey: 'created_on',
                                tooltip: 'When did the player connect first',
                                renderer: (obj) => {
                                    return (
                                        <Typography variant={'body1'}>
                                            {renderDateTime(obj.created_on)}
                                        </Typography>
                                    );
                                }
                            }
                        ]}
                        loading={false}
                        rows={players}
                        rowsPerPage={rowPerPageCount}
                        page={page}
                        showPager={true}
                        count={players.length}
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
                    />
                </Grid>
            </Grid>
        </Formik>
    );
};

export const AdminNetworkPage = () => {
    const [value, setValue] = React.useState(0);

    const handleChange = (_: React.SyntheticEvent, newValue: number) => {
        setValue(newValue);
    };

    return (
        <Grid container padding={0} spacing={2}>
            <Grid xs={9}>
                <ContainerWithHeader
                    title={'Network Tools'}
                    iconLeft={<LeakAddIcon />}
                >
                    <Box sx={{ borderBottom: 1, borderColor: 'divider' }}>
                        <Tabs
                            value={value}
                            onChange={handleChange}
                            aria-label="basic tabs example"
                        >
                            <Tab label="Find Players" />
                            <Tab label="IP Info" />
                            <Tab label={'External CIDR Bans'} />
                        </Tabs>
                    </Box>
                    <TabPanel value={value} index={0}>
                        <FindPlayerIP />
                    </TabPanel>
                    <TabPanel value={value} index={1}>
                        IPInfo
                    </TabPanel>
                    <TabPanel value={value} index={2}>
                        <NetworkBlockSources />
                    </TabPanel>
                </ContainerWithHeader>
            </Grid>
            <Grid xs={3}>
                <Stack spacing={2}>
                    <ContainerWithHeader
                        title={'Tool Overview'}
                        iconLeft={<HelpIcon />}
                    >
                        <List>
                            <ListItem>
                                <ListItemText
                                    primary={'Find Players'}
                                    secondary={`Query players using a particular ip or cidr range.`}
                                />
                            </ListItem>
                            <ListItem>
                                <ListItemText
                                    primary={'IP Info'}
                                    secondary={`Look up metadata for an ip/network`}
                                />
                            </ListItem>
                            <ListItem>
                                <ListItemText
                                    primary={'External CIDR Bans'}
                                    secondary={`Used for banning large range of address blocks using 3rd party URL sources. Response should be in the 
                                format of 1 cidr address per line. Invalid lines are discarded. Use the whitelist to override blocked addresses you want to allow.`}
                                />
                            </ListItem>
                        </List>
                    </ContainerWithHeader>
                    <ContainerWithHeader
                        title={'Blocked IP Checker'}
                        iconLeft={<VpnLockIcon />}
                    >
                        <NetworkBlockChecker />
                    </ContainerWithHeader>
                </Stack>
            </Grid>
        </Grid>
    );
};
