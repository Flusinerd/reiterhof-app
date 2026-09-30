-- Rain strength per blanket rule (JAN-84): a rule with rain = true can also require an amount of
-- rain over the horse's cover window, rain_min_mm <= rain_mm < rain_max_mm (NULL = open). The app
-- offers the steps of the weather card (light below 2 mm, moderate 2 to 8 mm, heavy from 8 mm) and
-- exact values.
ALTER TABLE blanket_rules
    ADD COLUMN rain_min_mm numeric,
    ADD COLUMN rain_max_mm numeric;
