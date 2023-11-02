import TextField from '@mui/material/TextField';
import React from 'react';
import { useTimer } from 'react-timer-hook';

export interface DelayedTextInputProps {
    delay?: number;
    onChange: (value: string) => void;
    placeholder: string;
    value: string;
    setValue: (value: string) => void;
    minLength?: number;
}

export const DelayedTextInput = ({
    delay,
    onChange,
    placeholder,
    value,
    setValue,
    minLength = 2
}: DelayedTextInputProps) => {
    const { restart } = useTimer({
        autoStart: false,
        expiryTimestamp: new Date(),
        onExpire: () => {
            onChange(value.length <= minLength ? '' : value);
        }
    });

    const onInputChange = (
        event: React.ChangeEvent<HTMLTextAreaElement | HTMLInputElement>
    ) => {
        setValue(event.target.value);
        const time = new Date();
        time.setSeconds(time.getSeconds() + (delay ?? 1));
        restart(time, true);
    };

    return (
        <TextField
            fullWidth
            value={value}
            placeholder={placeholder}
            onChange={onInputChange}
            error={value.length > 0 && value.length < minLength}
            helperText={
                value.length > 0 && value.length < minLength
                    ? `Minimum length: ${minLength}`
                    : ''
            }
        />
    );
};
