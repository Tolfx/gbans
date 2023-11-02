import Avatar from '@mui/material/Avatar';
import Box from '@mui/material/Box';
import Stack from '@mui/material/Stack';
import Typography from '@mui/material/Typography';
import { useTheme } from '@mui/material/styles';
import React, { MouseEventHandler } from 'react';
import { useNavigate } from 'react-router-dom';

export interface PersonCellProps {
    steam_id: string;
    personaname: string;
    avatar_hash: string;
    onClick?: MouseEventHandler | undefined;
}

export const PersonCell = ({
    steam_id,
    avatar_hash,
    personaname,
    onClick
}: PersonCellProps) => {
    const navigate = useNavigate();
    const theme = useTheme();

    return (
        <Stack
            direction={'row'}
            alignItems={'center'}
            onClick={
                onClick != undefined
                    ? onClick
                    : () => {
                          navigate(`/profile/${steam_id}`);
                      }
            }
            sx={{
                '&:hover': {
                    cursor: 'pointer',
                    backgroundColor: theme.palette.background.default
                }
            }}
        >
            <Avatar
                alt={personaname}
                src={`https://avatars.akamai.steamstatic.com/${avatar_hash}.jpg`}
                variant={'square'}
                sx={{ height: '32px', width: '32px' }}
            />
            <Box
                height={'100%'}
                alignContent={'center'}
                alignItems={'center'}
                display={'inline-block'}
                marginLeft={2}
            >
                <Typography variant={'body1'}>{personaname}</Typography>
            </Box>
        </Stack>
    );
};
