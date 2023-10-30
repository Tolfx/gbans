import React from 'react';
import { FormikHandlers, FormikState } from 'formik/dist/types';
import TextField from '@mui/material/TextField';
import * as yup from 'yup';

export const ASNumberFieldValidator = yup
    .number()
    .required()
    .positive()
    .integer();

export const ASNumberField = ({
    formik
}: {
    formik: FormikState<{
        as_num: number;
    }> &
        FormikHandlers;
}) => {
    return (
        <TextField
            type={'number'}
            fullWidth
            label={'Autonomous System Number'}
            id={'as_num'}
            name={'as_num'}
            value={formik.values.as_num}
            onChange={formik.handleChange}
            error={formik.touched.as_num && Boolean(formik.errors.as_num)}
            helperText={formik.touched.as_num && formik.errors.as_num}
        />
    );
};
