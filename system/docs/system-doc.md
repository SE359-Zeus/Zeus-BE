# Zeus ERP: Authentication Definition 

This document defines the Role-Based Access Control (RBAC) matrix and Authentication protocols for the Zeus microservices application.

---

## 1. Authentication Architecture (Kiến trúc xác thực)

The Zeus ecosystem employs a dual-method authentication strategy to secure both human interactions and system-to-system integrations.

### A. Token-Based Auth (Asymmetric - Bất đối xứng)
Used for user-facing applications (Web Dashboard).
- **Access Token (Mã truy cập):** A short-lived JWT generated using an **Asymmetric Algorithm** (e.g., RS256/EdDSA).
  - **Verification:** Each microservice validates the token locally using a **Public Key (Khóa công khai)**, eliminating the need for a synchronous call to the IAM service for every request.
- **Refresh Token (Mã làm mới):** A long-lived token used to rotate Access Tokens. Also signed asymmetrically to ensure integrity.

### B. API Key Authentication (Mã API)
Used for Zent call API from Zeus.
- **Exposure:** Currently, only the **SCM Service** is authorized to expose and manage API Keys.
- **Storage:** All valid API Keys are persisted in the **SCM database**.
- **Verification Logic:**
  - When a request contains an API Key (typically in the `X-API-KEY` header), the receiving service queries the **SCM database** to verify the key's validity and retrieve its authorized scopes.

---

## 2. Role Hierarchy Overview (Tổng quan phân cấp vai trò)

| Role Level | Description |
| :--- | :--- |
| **Administrator** | Global system control, user management, and employee oversight. |
| **Operator** | Functional management (Approvals, Priority Overrides, Sourcing Logic). |
| **Worker** | Execution of operational tasks (Data entry, Receiving, Order Ingestion). |

*Note: The HR module's functional roles (Operator/Worker) have been consolidated into the Global Administrator to streamline personnel management.*

---

## 3. Module-Specific Roles (Vai trò theo mô-đun)

### A. SCM (Supply Chain Management)
*Governs acquisition (upstream) and distribution (downstream).*

*   **SCM Operator:**
    *   **Responsibilities:** Resolve vendor selection (sourcing intelligence), approve Purchase Orders (POs), and orchestrate outbound dispatch.
    *   **Permissions:** 
        - `scm:vendor:resolve` (Ghi đè nhà cung cấp tối ưu)
        - `scm:po:approve` (Duyệt đơn mua hàng)
        - `scm:po:bulk_approve` (Duyệt hàng loạt)
        - `scm:supplier:manage` (Quản lý hồ sơ nhà cung cấp)
        - `scm:dispatch:orchestrate` (Điều phối lộ trình giao hàng)
*   **SCM Worker:**
    *   **Responsibilities:** Physical validation of inbound shipments, inventory ledger updates, and shipment packaging.
    *   **Permissions:** 
        - `scm:receipt:receive` (Nhập kho - Blind receiving)
        - `scm:receipt:inspect` (Kiểm tra chất lượng/Date code)
        - `scm:inventory:adjust` (Điều chỉnh tồn kho thực tế)
        - `scm:dispatch:pack` (Đóng gói và in nhãn)

### B. MRP (Material Requirement Planning)
*Governs production readiness and component hierarchies.*

*   **MRP Operator:**
    *   **Responsibilities:** Manage Component Catalog/BOM, trigger calculation runs, and execute SCM demand handoffs.
    *   **Permissions:** 
        - `mrp:catalog:manage` (Quản lý danh mục linh kiện & BOM)
        - `mrp:demand:trigger` (Kích hoạt tổng hợp nhu cầu vật tư)
        - `mrp:readiness:analyze` (Phân tích khả năng đáp ứng sản xuất)
        - `mrp:handoff:procure` (Chuyển yêu cầu mua hàng sang SCM)
*   **MRP Worker:**
    *   **Responsibilities:** Monitor material readiness and generate warehouse pick lists for the production floor.
    *   **Permissions:** 
        - `mrp:readiness:view` (Xem bảng cân đối vật tư)
        - `mrp:picklist:generate` (Tạo danh sách soạn hàng)
        - `mrp:ledger:read` (Xem nhật ký kho - Read-only)

### C. Sales (Order Management)
*Governs client ingestion and fulfillment priority.*

*   **Sales Operator:**
    *   **Responsibilities:** Manage Client Registry (Tier assignments), resolve fulfillment bottlenecks via priority overrides.
    *   **Permissions:** 
        - `sales:client:manage` (Quản lý hồ sơ/Tier khách hàng)
        - `sales:priority:override` (Ghi đè thứ tự ưu tiên phân bổ)
        - `sales:fulfillment:monitor` (Giám sát hàng đợi phân bổ)
*   **Sales Worker:**
    *   **Responsibilities:** API order validation, sales order creation, and tracking client fulfillment status.
    *   **Permissions:** 
        - `sales:order:create` (Tạo/Nhập đơn hàng)
        - `sales:order:track` (Theo dõi trạng thái đơn hàng)
        - `sales:client:read` (Xem thông tin khách hàng)

---

## 4. Global Administrative Roles

### **Administrator**
*   **Responsibilities:** The "Root" system user. Manages Identity (IAM), Employee records (formerly HR), and Global Config.
*   **Permissions:**
    - `iam:user:manage` (Quản lý người dùng)
    - `iam:role:assign` (Gán vai trò và quyền)
    - `hr:employee:manage` (Quản lý hồ sơ nhân viên & Chấm công)
    - `audit:log:read` (Truy xuất nhật ký hệ thống)
    - `system:config:write` (Cấu hình tham số hệ thống)

---


