CREATE UNIQUE INDEX patients_phone_number_normalized_uidx
ON patients ((regexp_replace(phone_number, '[^0-9]', '', 'g')));
