


CREATE TABLE banking_partners (
    id bigserial PRIMARY KEY,
    currency_code VARCHAR(3) NOT NULL
        REFERENCES currencies(code),
    country_code CHAR(2) NOT NULL,
    provider_name VARCHAR(300) NOT NULL,
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    detail_type VARCHAR(30) NOT NULL,

    CONSTRAINT banking_partners_unique
        UNIQUE (currency_code, country_code, provider_name),

    CONSTRAINT banking_partners_detail_type_check
        CHECK (
            detail_type IN (
                'IBAN',
                'ACCOUNT_NUMBER',
                'ROUTING_NUMBER'
            )
        )
);