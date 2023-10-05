import React, { JSX } from 'react';
import Stack from '@mui/material/Stack';
import Typography from '@mui/material/Typography';
import Box from '@mui/material/Box';

export const Footer = (): JSX.Element => {
    return (
        <Box
            sx={{
                textAlign: 'center',
                marginTop: '1rem',
                padding: '1rem',
                marginBottom: '0',
                height: '100%'
            }}
        >
            <Stack>
                <Typography variant={'subtitle2'} color={'text'}>
                    Copyright &copy; {window.gbans.site_name || 'gbans'}{' '}
                    {window.gbans.build_version} {new Date().getFullYear()}
                </Typography>
            </Stack>
        </Box>
    );
};
