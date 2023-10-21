import DoDisturbIcon from '@mui/icons-material/DoDisturb';
import Button from '@mui/material/Button';
import Link from '@mui/material/Link';
import Stack from '@mui/material/Stack';
import Typography from '@mui/material/Typography';
import Grid from '@mui/material/Unstable_Grid2';
import React, { useMemo } from 'react';
import SteamID from 'steamid';
import { handleOnLogin } from '../api';
import { ContainerWithHeader } from '../component/ContainerWithHeader';
import { useCurrentUserCtx } from '../contexts/CurrentUserCtx';
import steamLogo from '../icons/steam_login_lg.png';

export interface LoginFormProps {
    message?: string;
}

export const Login = ({ message }: LoginFormProps) => {
    const { currentUser } = useCurrentUserCtx();

    const loggedInUser = useMemo(() => {
        const sid = new SteamID(currentUser.steam_id);
        return sid.isValidIndividual();
    }, [currentUser.steam_id]);

    return (
        <Grid container justifyContent={'center'} alignItems={'center'}>
            <Grid xs={12}>
                <ContainerWithHeader
                    title={'Permission Denied'}
                    iconLeft={<DoDisturbIcon />}
                >
                    <>
                        {loggedInUser && (
                            <Typography variant={'body1'} padding={2}>
                                Insufficient permission to access this page.
                            </Typography>
                        )}
                        {!loggedInUser && (
                            <>
                                <Typography
                                    variant={'body1'}
                                    padding={2}
                                    paddingBottom={0}
                                >
                                    {message ??
                                        'To access this page, please login using your steam account below.'}
                                </Typography>
                                <Stack
                                    justifyContent="center"
                                    gap={2}
                                    flexDirection="row"
                                    width={1.0}
                                    flexWrap="wrap"
                                    padding={2}
                                >
                                    <Button
                                        sx={{ alignSelf: 'center' }}
                                        component={Link}
                                        href={handleOnLogin(
                                            window.location.pathname
                                        )}
                                    >
                                        <img
                                            src={steamLogo}
                                            alt={'Steam Login'}
                                        />
                                    </Button>
                                </Stack>
                            </>
                        )}
                    </>
                </ContainerWithHeader>
            </Grid>
        </Grid>
    );
};
