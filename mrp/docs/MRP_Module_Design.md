# Manufacturing Resource Planning (MRP) Business Architecture

## Overview
This document defines the Manufacturing Resource Planning (MRP) module for the Zeus ERP system. The module governs material readiness, component catalog management, and the aggregation of manufacturing demands. It acts as the intelligence layer that translates production schedules into technical component requirements and identifies material deficits prior to procurement.

## 1. Core Subsystems

### Component Catalog
The Component Catalog serves as the master record for all hardware parts and product assemblies. It maintains the technical specifications, unit costs, and structural dependencies required for build orchestration.

**Core Data Structures:**
- **Component Specification:** Detailed metadata for individual SKUs, including type (Raw Material, Assembly), unit cost, and functional description.
- **Product Assembly (BOM):** A hierarchical map linking a parent product (e.g., Workstation Alpha) to its constituent component SKUs and required quantities.
- **Dependency Mapping (Where Used):** A reverse-lookup table that identifies which parent assemblies are affected by a specific component SKU, enabling rapid impact analysis for shortages.

### Material Readiness Matrix
The Material Readiness Matrix is the primary dashboard for production controllers. It provides real-time visibility into build viability by correlating open production orders with current on-hand stock.

**Execution Mechanics:**
- **Readiness Status Logic:** The system evaluates each production order against the inventory snapshot to assign a readiness state:
    - **Clear to Build:** 100% of required components are allocated and available.
    - **Partial:** Some components are available, but critical items are missing.
    - **Shortage:** Critical components are unavailable, blocking production.
- **Deficit Breakdown:** For orders in "Shortage" or "Partial" states, the system generates a SKU-level breakdown of the deficit (Required vs. On-Hand).

### Inventory Ledger (Read-Only)
The Inventory Ledger provides a secure, immutable audit trail of all material movements within the system. Within the MRP module, this ledger is strictly read-only to ensure data integrity during readiness calculations.

**Data Attributes:**
- **Transaction Types:** Tracks Stock In (IN), Stock Out (OUT), and Adjustments (ADJ).
- **Running Balance:** Maintains a continuous calculation of available stock per SKU at specific locations (e.g., WH-A / Zone-C1).
- **Traceability:** Links every stock movement to an operator ID and a reference document (e.g., Production Order or Goods Receipt).

### Manufacturing Demand
The Manufacturing Demand subsystem aggregates component shortages across all open production orders. It simplifies the transition from "production requirement" to "procurement request."

**Execution Mechanics:**
- **Deficit Aggregation:** Instead of managing individual purchase orders, this subsystem pools all shortages into an aggregated manufacturing demand list.
- **SCM Handoff:** The aggregated demand serves as the primary data input for the SCM module. MRP identifies what is missing; SCM determines who to buy it from and executes the financial transaction.
- **Pick List Generation:** For "Clear to Build" orders, the system generates warehouse pick lists to initiate the physical movement of parts to the production floor.

## 2. System Interconnections
The MRP module functions as the bridge between production planning and supply chain execution. Interconnections are delineated by subsystem and external dependency.

### Component Catalog
**Data Inputs:**
- **From Engineering:** Hardware specifications and assembly structures (BOM).

**Data Outputs:**
- **To Readiness Matrix:** SKU definitions and quantity-per-build requirements.

### Material Readiness Matrix
**Data Inputs:**
- **From Sales/Production:** Open manufacturing orders and target build dates.
- **From Inventory Ledger:** Real-time on-hand stock levels.

**Data Outputs:**
- **To Manufacturing Demand:** Identified component shortages and quantities.

### Inventory Ledger
**Data Inputs:**
- **From Goods Receipt (SCM):** Verified inbound shipments (increments stock).
- **From Production Floor:** Consumed materials upon build completion (decrements stock).

**Data Outputs:**
- **To Financial Module:** Inventory valuation based on unit costs and stock levels.

### Manufacturing Demand
**Data Inputs:**
- **From Readiness Matrix:** Aggregated SKU-level deficits.

**Data Outputs:**
- **To SCM (Vendor Routing):** Quantitative component shortages to trigger procurement workflows.
- **To Warehouse:** Pick lists for material kitting and distribution.

## 3. Internal Interface Design
Provide an internal dashboard for production controllers and inventory managers. Adhere to the established UI paradigm: global dark-themed sidebar navigation, secondary horizontal tabs where necessary, KPI summary cards, and full-width interactive data tables.

### View 1: Dashboard (Material Readiness Matrix)
Serve as the primary monitoring interface for production viability and deficit detection.
- **Purpose:** Provide real-time build visibility, highlight blocked production orders, and seamlessly bridge deficits to the procurement pipeline.
- **Layout Structure:**
    - **Filter Toolbar:** Provide filters for open orders and a global search input for SKUs or Order IDs.
    - **Data Table:** Render a full-width grid. Define columns for Order ID, Target Build, Qty, Readiness Status, Missing Components, and Actions.
    - **Global Actions:** Provide a "Draft Purchase Orders" primary button at the view level.
- **Visual Cues:** Utilize distinct color-coded pill badges for state visibility: SHORTAGE (red) and CLEAR TO BUILD (green).
- **Interaction:** Clicking a table row expands an accordion panel revealing the "Component Deficit Breakdown." This sub-table displays Component SKU, Required, On-Hand, and Shortage metrics. Render a contextual "Generate PO for Deficits" action link within this expanded state to trigger the SCM demand handoff.

### View 2: BOM & Catalog
Serve as the master data management interface for hardware specifications and structural build requirements.
- **Purpose:** Define product assemblies, manage raw material metadata, and trace component dependencies.
- **Layout Structure:**
    - **Split-Pane Layout:** Divide the view into two proportional columns.
    - **Left Pane (Product Assemblies):** Render a hierarchical tree view listing all parent assemblies (e.g., "Workstation Alpha"). Expandable nodes display required child SKUs and quantities.
    - **Right Pane (Component Specification):** Render a detailed property panel for the selected SKU. Display unit cost, hardware descriptions, and a "Where Used (Dependencies)" table listing all parent assemblies reliant on the component.
- **Interaction:** Provide a global "Create Product Assembly" button. Selecting any node in the left pane dynamically updates the right pane with the specific component or assembly metadata.

### View 3: Inventory Ledger
Serve as the read-only audit trail for physical stock movements affecting production calculations.
- **Purpose:** Trace historical stock changes, verify incoming SCM receipts, and audit manual adjustments.
- **Layout Structure:**
    - **KPI Header:** Render metric cards detailing Total SKUs, Stock In (Today), Stock Out (Today), and Adjustments.
    - **Filter Toolbar:** Render horizontal tabs to pivot the table view (All, Stock In, Stock Out, Adjustments).
    - **Data Table:** Render a full-width grid. Define columns for Txn ID, SKU, Type, Qty Change, Balance, Location, Timestamp, and Operator.
- **Visual Cues:** Color-code the Type and Qty Change columns (Green for +Stock In, Red for -Stock Out, Amber for Adjustments) to enable rapid visual auditing.
