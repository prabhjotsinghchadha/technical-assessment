-- Customer Status
INSERT INTO customer_status (status_name, description) VALUES 
('Active', 'Customer is currently active and in good standing'),
('Inactive', 'Customer is not currently active'),
('Suspended', 'Customer account is temporarily suspended'),
('Closed', 'Customer account is permanently closed'),
('Pending', 'Customer application is pending approval');

-- Customer Type
INSERT INTO customer_type (type_name, description) VALUES 
('Individual', 'An individual customer'),
('Business', 'A business customer'),
('VIP', 'A very important customer with special privileges'),
('Non-Profit', 'A non-profit organization customer'),
('Government', 'A government entity customer');


-- Account Status
INSERT INTO account_status (status_name, description) VALUES
('Active', 'The account is open and fully functional.'),
('Inactive', 'The account has not been used for a specified period but is still open.'),
('Dormant', 'The account has been inactive for a longer period and may require specific actions to reactivate.'),
('Closed', 'The account has been permanently closed and cannot be used for transactions.'),
('Suspended', 'The account is temporarily restricted from performing transactions.'),
('Pending', 'The account is in the process of being opened or closed.'),
('Overdrawn', 'The account balance is negative due to overdrawing funds.'),
('Frozen', 'The account is restricted due to legal or regulatory actions.'),
('Restricted', 'The account has specific limitations placed on it.'),
('Verified', 'The account has been verified through a KYC process and is fully functional.'),
('Unverified', 'The account has not yet completed the KYC process and may have limitations on its usage.');


-- Account Type
INSERT INTO account_type (type_name, description) VALUES
('Savings', 'A savings account with interest accrual.'),
('Checking', 'A checking account for daily transactions.'),
('Business', 'An account designed for business use.'),
('Student', 'A student account with educational benefits.'),
('Joint', 'An account shared by two or more individuals.');


-- Transaction Status
INSERT INTO transaction_status (status_name, description) VALUES 
('Pending', 'Transaction is pending processing'),
('Completed', 'Transaction has been completed successfully'),
('Aproved', 'Transaction has been approved by the system'),
('Failed', 'Transaction has failed'),
('Pending Approval', 'Transaction requires approval from an administrator or supervisor'),
('On Hold', 'Transaction is temporarily suspended or on hold for further review'),
('Expired', 'Transaction has expired and cannot be processed'),
('Cancelled', 'Transaction was cancelled before processing'),
('Refunded', 'Transaction amount was refunded to the customer');


-- Transaction Type
INSERT INTO transaction_type (type_name, description) VALUES 
('Deposit', 'Deposit transaction'),
('Withdrawal', 'Withdrawal transaction'),
('Transfer', 'Transfer transaction'),
('Payment', 'Payment transaction'),
('Purchase', 'Purchase transaction'),
('Salary', 'Salary deposit transaction'),
('Expense', 'Expense withdrawal transaction'),
('Loan', 'Loan transaction'),
('Interest', 'Interest transaction');


-- Demo customers
INSERT INTO customers (customer_type, customer_status, customer_name, gender, identification_number, email, phone, address)
VALUES
    (1, 1, 'John Doe', 'Male', 'ID1000001', 'john.doe@example.com', '5551000001', '100 Market Street'),
    (1, 1, 'Jane Smith', 'Female', 'ID1000002', 'jane.smith@example.com', '5551000002', '200 River Avenue');

-- Demo accounts: two active accounts plus one inactive account
INSERT INTO accounts (customer_id, account_type, account_status, account_number, iban, holder_name, balance, currency)
VALUES
    (1, 1, 1, '1000000001', 'US0010000000010000001', 'John Doe', 10000.00, 'USD'),
    (1, 2, 1, '1000000002', 'US0010000000020000002', 'John Doe', 500.00, 'USD'),
    (2, 1, 1, '2000000001', 'US0020000000010000001', 'Jane Smith', 2500.00, 'USD'),
    (2, 1, 2, '2000000002', 'US0020000000020000002', 'Jane Smith', 100.00, 'USD');

