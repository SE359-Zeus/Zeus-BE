# Sales Business Architecture

## Overview
This document defines the Sales module for the Zeus ERP system. The module governs B2B and B2C order ingestion, priority-aware inventory allocation, and customer-facing fulfillment tracking.

## 1. Core Subsystems

### Order Management
Ingest external client demand via headless API endpoints. Do not implement a consumer-facing storefront UI. Act strictly as the backend order gateway for B2B and B2C clients.

**Core Data Structures:**
- **Client Profile:** A minimal registry record used exclusively to route physical goods, map order histories, and dictate fulfillment priority. Stores Client ID, Name, Default Destination Address, and a binary Tier Assignment (B2B or B2C).
- **Sales Order:** The primary transaction document. Stores Order ID, Client ID, required delivery date, and overall order status.
- **Sales Order Item:** Line items attached to the Sales Order. Maps directly to finished goods SKUs in the Product catalog. Tracks requested quantity, allocated quantity, and unit price.

**Execution Mechanics:**
- **API Ingestion & Validation:**
    - Receive standard JSON order payloads via the `/api/v1/sales/orders` endpoint containing client details, requested SKUs, quantities, and optional destination overrides.
    - Validate requested SKUs against the Product catalog via synchronous internal RPC to ensure they represent active, orderable finished goods. Reject payloads containing invalid or raw-material SKUs.
    - Validate requested quantities against minimum/maximum order limits.
- **Client Resolution:**
    - Extract the client identifier from the inbound payload and query the Client Profile registry.
    - If the client exists, link the new Sales Order to the existing Client ID.
    - If the client does not exist, automatically generate a new Client Profile using the provided name, shipping details, and a default Tier Assignment (B2C) before linking the order.
- **Order State Management:** Enforce a strict, one-way state machine for the order lifecycle:
    - **Pending (Unallocated):** Initial state upon successful API ingestion. The order is logged, but no physical inventory is reserved. Fully editable via API.
    - **Processing (Allocated):** Triggered by the Fulfillment Orchestration subsystem. Physical inventory is reserved. Locked for external editing.
    - **Delivering (Dispatched):** Triggered by SCM Downstream Logistics. The physical hardware has left the warehouse.
    - **Completed:** Triggered via SCM carrier webhook or manual API confirmation indicating successful client receipt.
    - **Cancelled:** Terminal state. Drops the order and releases any allocated inventory back to the available Product pool.
- **Concurrency Control (Modification Lock):** Prevent race conditions between client-initiated API modifications and internal fulfillment processes.
    - Apply an atomic read/write lock to the Sales Order record the moment the Fulfillment Orchestrator queries it for inventory allocation.
    - If a client submits a `PATCH /api/v1/sales/orders/[id]` or `DELETE` request while the lock is active or after the state has transitioned to Processing, reject the request with a `409 Conflict` error, forcing the client to initiate a post-dispatch return process.

### Fulfillment Orchestration
Evaluate incoming orders against available inventory, assign fulfillment priority, and bind physical stock to specific client demands. This subsystem acts as the internal traffic controller bridging Sales and SCM.

**Core Data Structures:**
- **Allocation Queue:** An in-memory or Redis-backed sorted set that orders Pending Sales Orders based on calculated priority scores.
- **Inventory Reservation:** A cross-module transactional record that temporarily locks a specific quantity of a finished good SKU in the Product module, preventing it from being allocated to subsequent orders.

**Execution Mechanics:**
- **Priority Queue Sorting:** Periodically sweep the database for Pending Sales Orders and sort them into the Allocation Queue using deterministic logic:
    1. **Tier Assignment:** Query the Client Profile linked to the order to extract the binary tier. Execute B2B allocations strictly before B2C orders.
    2. **Delivery Date:** Within the same tier, sort by the nearest required delivery date.
    3. **FIFO Fallback:** If tier and delivery date are identical, execute based on the original API ingestion timestamp.
- **Cross-Module Availability Check:**
    - For the top order in the queue, issue a synchronous RPC call to the Product module to query the "Available to Promise" (ATP) balance for the required finished goods SKUs.
    - **Partial Fulfillment Logic:** If ATP covers only a fraction of the order, halt allocation for that specific order until stock replenishes, or execute a partial allocation depending on the client's configured shipping preferences.
- **Atomic Stock Reservation:**
    - Once availability is confirmed, issue an atomic `RESERVE` command to the Product module.
    - This action decrements the ATP balance in the Product module (soft lock) without yet dispatching the physical goods from the warehouse.
    - If the RPC `RESERVE` command fails (e.g., due to a race condition where inventory was manually adjusted via cycle count), abort the allocation for that order and requeue it.
- **SCM Dispatch Trigger:**
    - Upon successful reservation, transition the Sales Order state to **Processing**.
    - Generate a fulfillment manifest containing the Order ID, SKUs, reserved quantities, and destination address.
    - Transmit this manifest to the SCM Downstream Logistics subsystem to trigger the physical warehouse picking and packing workflow.

## 2. System Interconnections
The Sales module functions as the bridge between external client demand and internal logistics execution. Interconnections are delineated by subsystem and external dependency.

### Order Management
**Data Inputs:**
- **From Clients:** Raw JSON order payloads, shipping destination overrides, and API cancellation/modification requests.
- **From Product/BOM Catalog:** Active finished-goods SKU definitions and catalog constraints to validate inbound client requests.

**Data Outputs:**
- **To Clients:** HTTP response codes, order confirmation IDs, and real-time order status views (Pending, Processing, Delivering, Completed).

### Fulfillment Orchestration
**Data Inputs:**
- **From Product/Inventory Ledger:** Real-time "Available to Promise" (ATP) finished goods balances to inform the Allocation Queue logic.
- **From SCM (Downstream Logistics):** Real-time tracking IDs, carrier assignments, and final delivery confirmation webhooks to progress the Order State Management tracker.

**Data Outputs:**
- **To Product/Inventory Ledger:** Atomic `RESERVE` commands to securely lock physical stock prior to warehouse dispatch.
- **To SCM (Downstream Logistics):** Finalized fulfillment manifests containing SKUs, reserved quantities, and shipping priorities to trigger physical warehouse operations.
- **To Financial Module (Phase 3):** Realized revenue data, locked transaction totals, and billing metadata generated upon transitioning an order to the Processing/Delivering states.

## 3. Internal Interface Design
Provide an internal dashboard for sales operators and fulfillment managers. Adhere to the established UI paradigm: dark-themed layout, top-level horizontal tab navigation within the main view, KPI summary cards, and full-width interactive data tables.

### Tab 1: Sales Orders
Serve as the primary monitoring interface for all inbound client demands.
- **Purpose:** Track the complete lifecycle of B2B and B2C orders, monitor state transitions, and resolve concurrency locks.
- **Layout Structure:**
    - **KPI Header:** Render three metric cards detailing Total Pending Orders, Active Processing Value, and 24-Hour Completed Orders.
    - **Filter Toolbar:** Provide a multi-select dropdown for order states, a date range picker for delivery targets, and a text input for Client ID searches.
    - **Data Table:** Render a full-width grid. Define columns for Order ID, Client Name, Required Date, Total Value, and Status.
- **Visual Cues:** Utilize distinct color-coded pill badges for state visibility: Pending (grey); Processing (blue); Delivering (yellow); and Completed (green).
- **Interaction:** Clicking an order row opens a right-side slide-out panel. The panel displays exact SKU line items, destination addresses, and current lock status, rendering read-only if the Fulfillment Orchestrator holds the allocation lock.

### Tab 2: Fulfillment Queue
Expose the internal logic of the Allocation Queue to allow manual oversight and emergency reprioritization.
- **Purpose:** Monitor real-time inventory allocation, identify fulfillment bottlenecks caused by stock deficits, and execute manual priority overrides.
- **Layout Structure:**
    - **ATP Summary Row:** Render a compact horizontal strip displaying the current "Available to Promise" levels for the top five highest-velocity finished goods.
    - **Priority List:** Render a vertically stacked list view rather than a standard table, emphasizing the sequential execution order. Group items visually by Tier Assignment headers ("B2B Priority" and "B2C Standard").
    - **Line Item Metrics:** Display the requested SKUs, current ATP deficit blocking the order (if applicable), and the original ingestion timestamp.
- **Interaction:** Render drag-and-drop handles on rows to permit manual priority overrides by authorized managers. Include a "Force Allocation" button on rows blocked by partial deficits to bypass the standard queue wait and trigger partial fulfillment logic.

### Tab 3: Client Registry
Manage the minimal client records necessary for order linking and downstream SCM logistics routing.
- **Purpose:** Audit auto-generated client profiles, correct shipping destination errors, modify tier assignments, and view historical order volume per client.
- **Layout Structure:**
    - **Data Table:** Render a standard grid. Define columns for Client ID, Name, Tier (B2B/B2C), Default Destination Address, and Total Lifetime Orders.
    - **Action Column:** Provide an "Edit" icon button.
- **Interaction:** Clicking the edit button opens a centered modal dialog containing text inputs.
