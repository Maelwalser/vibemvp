# SQL Server Skill Guide

## Overview

Microsoft SQL Server (T-SQL) advanced patterns: recursive CTEs with MAXRECURSION, window functions, MERGE for upsert, SEQUENCE and IDENTITY for key generation, partition functions for large tables, Always Encrypted for column-level encryption, temporal tables for history, and execution plan hints.

---

## T-SQL CTEs with OPTION(MAXRECURSION)

### Standard CTE

```sql
WITH regional_totals AS (
    SELECT
        region,
        SUM(amount)  AS total,
        COUNT(*)     AS order_count,
        AVG(amount)  AS avg_order
    FROM orders
    WHERE status = 'completed'
    GROUP BY region
)
SELECT r.region, r.total, r.order_count,
       r.total * 100.0 / SUM(r.total) OVER () AS pct_of_total
FROM regional_totals r
ORDER BY r.total DESC;
```

### Recursive CTE (with MAXRECURSION)

Default recursion limit is 100; override with `OPTION(MAXRECURSION n)`.

```sql
WITH RECURSIVE_ORG AS (
    -- Anchor
    SELECT
        EmployeeID,
        ManagerID,
        Name,
        0              AS Level,
        CAST(Name AS NVARCHAR(MAX)) AS OrgPath
    FROM Employees
    WHERE ManagerID IS NULL

    UNION ALL

    -- Recursive member
    SELECT
        e.EmployeeID,
        e.ManagerID,
        e.Name,
        r.Level + 1,
        r.OrgPath + N' > ' + e.Name
    FROM Employees e
    INNER JOIN RECURSIVE_ORG r ON r.EmployeeID = e.ManagerID
)
SELECT EmployeeID, Name, Level, OrgPath
FROM RECURSIVE_ORG
ORDER BY OrgPath
OPTION (MAXRECURSION 500);   -- Override default limit of 100; 0 = unlimited
```

---

## Window Functions

```sql
-- ROW_NUMBER, RANK, DENSE_RANK
SELECT
    OrderID,
    CustomerID,
    Amount,
    ROW_NUMBER()  OVER (PARTITION BY CustomerID ORDER BY OrderDate DESC) AS rn,
    RANK()        OVER (PARTITION BY CustomerID ORDER BY Amount DESC)    AS rnk,
    DENSE_RANK()  OVER (PARTITION BY CustomerID ORDER BY Amount DESC)    AS dense_rnk
FROM Orders;

-- LAG / LEAD
SELECT
    SaleDate,
    Revenue,
    LAG(Revenue, 1, 0) OVER (ORDER BY SaleDate)  AS PrevDayRevenue,
    LEAD(Revenue, 1, 0) OVER (ORDER BY SaleDate) AS NextDayRevenue,
    Revenue - LAG(Revenue, 1, 0) OVER (ORDER BY SaleDate) AS DayOverDay
FROM DailyRevenue;

-- Running total
SELECT
    TransactionDate,
    Amount,
    SUM(Amount) OVER (ORDER BY TransactionDate
                      ROWS BETWEEN UNBOUNDED PRECEDING AND CURRENT ROW) AS RunningTotal
FROM Transactions;

-- PERCENTILE_CONT (continuous interpolation)
SELECT
    PERCENTILE_CONT(0.5) WITHIN GROUP (ORDER BY Salary) OVER (PARTITION BY DeptID) AS MedianSalary
FROM Employees;

-- NTILE: quartile bucketing
SELECT
    CustomerID,
    LifetimeValue,
    NTILE(4) OVER (ORDER BY LifetimeValue DESC) AS Quartile
FROM Customers;
```

---

## MERGE for Upsert

```sql
MERGE INTO inventory AS target
USING (VALUES (@ProductID, @Quantity, @UpdatedAt)) AS source (ProductID, Quantity, UpdatedAt)
    ON target.ProductID = source.ProductID
WHEN MATCHED THEN
    UPDATE SET
        target.Quantity  = source.Quantity,
        target.UpdatedAt = source.UpdatedAt
WHEN NOT MATCHED BY TARGET THEN
    INSERT (ProductID, Quantity, UpdatedAt)
    VALUES (source.ProductID, source.Quantity, source.UpdatedAt)
WHEN NOT MATCHED BY SOURCE AND target.UpdatedAt < DATEADD(DAY, -30, GETUTCDATE()) THEN
    DELETE
OUTPUT $action AS MergeAction, inserted.ProductID;
```

Note: MERGE can cause race conditions in high-concurrency scenarios — consider wrapping in a transaction or using `WITH (HOLDLOCK)`:
```sql
MERGE INTO inventory WITH (HOLDLOCK) AS target ...
```

---

## SEQUENCE and IDENTITY

### IDENTITY (column-level)

```sql
CREATE TABLE Orders (
    OrderID    INT          IDENTITY(1,1) PRIMARY KEY,  -- start=1, increment=1
    CustomerID INT          NOT NULL,
    CreatedAt  DATETIME2    DEFAULT GETUTCDATE()
);

-- Get last inserted identity
SELECT SCOPE_IDENTITY() AS NewOrderID;
-- Or: OUTPUT INSERTED.OrderID in the INSERT statement
INSERT INTO Orders (CustomerID) OUTPUT INSERTED.OrderID VALUES (42);
```

### SEQUENCE (schema-level, shareable across tables)

```sql
-- Create sequence
CREATE SEQUENCE dbo.OrderNumberSeq
    AS BIGINT
    START WITH 10000
    INCREMENT BY 1
    MINVALUE 10000
    NO MAXVALUE
    CACHE 50;    -- Cache 50 values in memory for performance

-- Use in INSERT
INSERT INTO Orders (OrderID, CustomerID)
VALUES (NEXT VALUE FOR dbo.OrderNumberSeq, 42);

-- Use as column default
CREATE TABLE Invoices (
    InvoiceID BIGINT DEFAULT (NEXT VALUE FOR dbo.OrderNumberSeq) PRIMARY KEY,
    ...
);

-- Current value
SELECT current_value FROM sys.sequences WHERE name = 'OrderNumberSeq';
```

---

## Partition Functions

Partitioning splits a large table into smaller, manageable filegroups based on a column value range.

```sql
-- 1. Create partition function (defines boundary values)
CREATE PARTITION FUNCTION pf_OrderDate (DATE)
AS RANGE RIGHT FOR VALUES (
    '2022-01-01', '2023-01-01', '2024-01-01', '2025-01-01'
);
-- RANGE RIGHT: boundary value belongs to the right (newer) partition

-- 2. Create partition scheme (maps partitions to filegroups)
CREATE PARTITION SCHEME ps_OrderDate
AS PARTITION pf_OrderDate
TO ([FG_2021], [FG_2022], [FG_2023], [FG_2024], [FG_2025]);

-- 3. Create partitioned table
CREATE TABLE Orders (
    OrderID   INT      NOT NULL,
    OrderDate DATE     NOT NULL,
    Amount    DECIMAL(10,2),
    CONSTRAINT PK_Orders PRIMARY KEY (OrderID, OrderDate)
) ON ps_OrderDate (OrderDate);

-- 4. Partition-aligned index
CREATE INDEX IX_Orders_CustomerID ON Orders (CustomerID)
ON ps_OrderDate (OrderDate);

-- 5. Switch partition for fast archival (O(1) metadata operation)
ALTER TABLE Orders SWITCH PARTITION 1 TO Orders_Archive PARTITION 1;

-- Check partition sizes
SELECT
    p.partition_number,
    p.rows,
    rv.value AS boundary_value
FROM sys.partitions p
JOIN sys.partition_range_values rv ON rv.function_id = p.function_id
    AND rv.boundary_id = p.partition_number - 1
WHERE OBJECT_NAME(p.object_id) = 'Orders';
```

---

## Always Encrypted (Column-Level Encryption)

Always Encrypted encrypts data client-side; SQL Server never sees plaintext.

```sql
-- 1. Create Column Master Key (CMK) — references key in Windows Certificate Store or Azure Key Vault
CREATE COLUMN MASTER KEY CMK_1
WITH (
    KEY_STORE_PROVIDER_NAME = 'AZURE_KEY_VAULT',
    KEY_PATH = 'https://myvault.vault.azure.net/keys/CMK1/abc123'
);

-- 2. Create Column Encryption Key (CEK) — encrypted by CMK
CREATE COLUMN ENCRYPTION KEY CEK_1
WITH VALUES (
    COLUMN_MASTER_KEY = CMK_1,
    ALGORITHM = 'RSA_OAEP',
    ENCRYPTED_VALUE = 0x...  -- generated by SSMS/PowerShell wizard
);

-- 3. Define encrypted columns
CREATE TABLE Patients (
    PatientID    INT PRIMARY KEY,
    -- Deterministic: supports equality lookups (WHERE SSN = @ssn)
    SSN          NCHAR(11) ENCRYPTED WITH (
                     COLUMN_ENCRYPTION_KEY = CEK_1,
                     ENCRYPTION_TYPE = DETERMINISTIC,
                     ALGORITHM = 'AEAD_AES_256_CBC_HMAC_SHA_256'),
    -- Randomized: more secure, no equality lookups
    DateOfBirth  DATE ENCRYPTED WITH (
                     COLUMN_ENCRYPTION_KEY = CEK_1,
                     ENCRYPTION_TYPE = RANDOMIZED,
                     ALGORITHM = 'AEAD_AES_256_CBC_HMAC_SHA_256')
);
```

Connection string (enables client-side encryption/decryption):
```
Server=myserver;Database=mydb;Column Encryption Setting=Enabled;Authentication=Active Directory Integrated;
```

---

## Temporal Tables (System-Versioned)

Temporal tables automatically track row history with start/end timestamps.

```sql
-- Create temporal table
CREATE TABLE Employees (
    EmployeeID     INT          NOT NULL PRIMARY KEY,
    Name           NVARCHAR(100) NOT NULL,
    Salary         DECIMAL(10,2) NOT NULL,
    DepartmentID   INT          NOT NULL,
    -- System-time columns (managed by SQL Server)
    ValidFrom      DATETIME2    GENERATED ALWAYS AS ROW START NOT NULL,
    ValidTo        DATETIME2    GENERATED ALWAYS AS ROW END   NOT NULL,
    PERIOD FOR SYSTEM_TIME (ValidFrom, ValidTo)
)
WITH (SYSTEM_VERSIONING = ON (HISTORY_TABLE = dbo.EmployeesHistory));

-- Normal DML works as usual; history is auto-maintained
UPDATE Employees SET Salary = 85000 WHERE EmployeeID = 42;

-- Query current state
SELECT * FROM Employees WHERE EmployeeID = 42;

-- Query state at a point in time
SELECT * FROM Employees
FOR SYSTEM_TIME AS OF '2024-06-01 00:00:00'
WHERE EmployeeID = 42;

-- Query all versions in a range
SELECT * FROM Employees
FOR SYSTEM_TIME BETWEEN '2024-01-01' AND '2024-12-31'
WHERE EmployeeID = 42
ORDER BY ValidFrom;

-- Audit: who changed salary in last 30 days
SELECT e.EmployeeID, e.Name, e.Salary, e.ValidFrom, e.ValidTo
FROM Employees FOR SYSTEM_TIME ALL AS e
WHERE e.EmployeeID = 42
  AND e.ValidFrom >= DATEADD(DAY, -30, GETUTCDATE())
ORDER BY e.ValidFrom DESC;
```

---

## Execution Plan Hints

```sql
-- Force specific index
SELECT * FROM Orders WITH (INDEX(IX_Orders_CustomerID))
WHERE CustomerID = 42;

-- Force join type
SELECT o.OrderID, c.Name
FROM Orders o
INNER HASH JOIN Customers c ON c.CustomerID = o.CustomerID;   -- HASH JOIN
-- Or: INNER LOOP JOIN (nested loop), INNER MERGE JOIN

-- Recompile (generate fresh plan, avoid parameter sniffing)
SELECT * FROM Orders WHERE CustomerID = @CustomerID
OPTION (RECOMPILE);

-- Force max degree of parallelism
SELECT * FROM LargeTable
OPTION (MAXDOP 4);   -- 0 = unlimited, 1 = no parallelism

-- Optimize for specific parameter value (parameter sniffing fix)
SELECT * FROM Orders WHERE CustomerID = @CustomerID
OPTION (OPTIMIZE FOR (@CustomerID = 99));

-- Optimize for unknown (average plan)
SELECT * FROM Orders WHERE CustomerID = @CustomerID
OPTION (OPTIMIZE FOR UNKNOWN);

-- Force estimated row count
SELECT * FROM Orders WHERE Status = @Status
OPTION (USE HINT ('FORCE_LEGACY_CARDINALITY_ESTIMATION'));

-- CTE MAXRECURSION
WITH RecursiveCTE AS (...)
SELECT * FROM RecursiveCTE
OPTION (MAXRECURSION 0);   -- 0 = unlimited
```

---

## Key Rules

- Use `SEQUENCE` over `IDENTITY` when sharing key generation across multiple tables or when pre-fetching keys in application code
- `MERGE` requires `WITH (HOLDLOCK)` in concurrent workloads to prevent race conditions between the check and the insert/update
- Partition function `RANGE RIGHT` is typically correct for date-based partitioning (the boundary date belongs to the newer partition)
- Always Encrypted Deterministic mode is required for WHERE clause lookups; Randomized mode provides stronger security but limits query patterns
- Temporal tables retain full history — implement retention policy by disabling system versioning, truncating history, and re-enabling
- `OPTION(MAXRECURSION 0)` disables the limit entirely — use with caution; set a meaningful limit instead
- Window functions with `ROWS` frame are deterministic; `RANGE` frame can include duplicates unexpectedly
- Use `SCOPE_IDENTITY()` not `@@IDENTITY` — `@@IDENTITY` includes identity values from triggers on the same connection
