import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import {
  Activity,
  AlertTriangle,
  Boxes,
  CreditCard,
  Gauge,
  ListChecks,
  Menu,
  RefreshCw,
  Server,
  Settings,
  Shield,
  Users,
  WalletCards,
} from "lucide-react";
import type { FormEvent, ReactNode } from "react";
import type { LucideIcon } from "lucide-react";
import { useState } from "react";
import { Button } from "@/components/ui/button";

type ListResponse<T> = { items: T[]; total: number };
type User = { id: string; email: string; status: string; created_at: string };
type PaymentOrder = { id: string; user_id: string; amount_cents: number; status: string; provider_trade_no: string; created_at: string };
type Ledger = { id: string; user_id: string; type: string; amount_cents: number; balance_after_cents: number; held_after_cents: number; reference_id: string; created_at: string };
type Product = { id: string; name: string; ip_type: string; enabled: boolean };
type ProductPrice = { protocol: string; duration_days: number; unit_cents: number };
type Inventory = { id: string; line_id: string; node_id: string; ip: string; port: number; protocols: string[]; status: string; manual_hold: boolean; compliance_hold: boolean };
type Order = { id: string; user_id: string; proxy_account_id: string; protocol: string; amount_cents: number; status: string; failure_reason?: string; created_at: string };
type ProxyAccount = { id: string; user_id: string; node_id: string; listen_ip: string; port: number; username: string; status: string; lifecycle_status: string; expires_at: string };
type NodeSummary = { id: string; code: string; status: string; last_online_at: string; capabilities?: string[] };

const baseURL = import.meta.env.VITE_API_BASE_URL ?? "";
const money = (cents: number) => `¥${(cents / 100).toFixed(2)}`;

async function apiJSON<T>(path: string, init?: RequestInit): Promise<T> {
  const response = await fetch(`${baseURL}${path}`, {
    credentials: "include",
    ...init,
    headers: { "Content-Type": "application/json", ...init?.headers },
  });
  if (!response.ok) {
    const text = await response.text();
    throw new Error(text || `请求失败: ${response.status}`);
  }
  return response.json() as Promise<T>;
}

export function App() {
  const queryClient = useQueryClient();
  const [username, setUsername] = useState("admin");
  const [password, setPassword] = useState("rayip-admin");
  const [inventoryForm, setInventoryForm] = useState({
    line_id: "dev-line",
    node_id: "local-home-001",
    ip: "203.0.113.10",
    port: "18080",
    protocols: "SOCKS5,HTTP",
    status: "AVAILABLE",
  });

  const login = useMutation({
    mutationFn: () => apiJSON("/api/admin/auth/login", { method: "POST", body: JSON.stringify({ username, password }) }),
    onSuccess: () => void queryClient.invalidateQueries(),
  });

  const users = useQuery({ queryKey: ["admin-users"], queryFn: () => apiJSON<ListResponse<User>>("/api/admin/users"), retry: false });
  const payments = useQuery({ queryKey: ["payment-orders"], queryFn: () => apiJSON<ListResponse<PaymentOrder>>("/api/admin/payment-orders"), enabled: users.isSuccess });
  const ledger = useQuery({ queryKey: ["wallet-ledger"], queryFn: () => apiJSON<ListResponse<Ledger>>("/api/admin/wallet-ledger"), enabled: users.isSuccess });
  const products = useQuery({ queryKey: ["products"], queryFn: () => apiJSON<{ items: Product[]; prices: ProductPrice[]; total: number }>("/api/admin/products"), enabled: users.isSuccess });
  const inventory = useQuery({ queryKey: ["inventory"], queryFn: () => apiJSON<ListResponse<Inventory>>("/api/admin/inventory"), enabled: users.isSuccess });
  const orders = useQuery({ queryKey: ["orders"], queryFn: () => apiJSON<ListResponse<Order>>("/api/admin/orders"), enabled: users.isSuccess });
  const proxies = useQuery({ queryKey: ["proxies"], queryFn: () => apiJSON<ListResponse<ProxyAccount>>("/api/admin/proxies"), enabled: users.isSuccess });
  const nodes = useQuery({ queryKey: ["nodes"], queryFn: () => apiJSON<ListResponse<NodeSummary>>("/api/admin/nodes"), enabled: users.isSuccess });

  const addInventory = useMutation({
    mutationFn: () =>
      apiJSON("/api/admin/inventory", {
        method: "POST",
        body: JSON.stringify({
          ...inventoryForm,
          port: Number(inventoryForm.port),
          protocols: inventoryForm.protocols.split(",").map((item) => item.trim()).filter(Boolean),
        }),
      }),
    onSuccess: () => void queryClient.invalidateQueries({ queryKey: ["inventory"] }),
  });
  const retryOrder = useMutation({
    mutationFn: (orderID: string) => apiJSON(`/api/admin/orders/${orderID}/retry-fulfillment`, { method: "POST" }),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ["orders"] });
      void queryClient.invalidateQueries({ queryKey: ["proxies"] });
      void queryClient.invalidateQueries({ queryKey: ["inventory"] });
    },
  });
  const reconcileProxy = useMutation({
    mutationFn: (proxyID: string) => apiJSON(`/api/admin/proxies/${proxyID}/reconcile`, { method: "POST" }),
    onSuccess: () => void queryClient.invalidateQueries({ queryKey: ["proxies"] }),
  });

  if (users.isError) {
    return (
      <div className="grid min-h-screen place-items-center bg-[#f3f4f6] px-6 text-[#1f2430]">
        <form
          className="w-full max-w-[420px] rounded-lg bg-white p-7 shadow-[0_12px_36px_rgba(15,23,42,0.08)]"
          onSubmit={(event: FormEvent) => {
            event.preventDefault();
            login.mutate();
          }}
        >
          <div className="mb-7 flex items-center gap-3">
            <div className="grid size-10 place-items-center rounded-md bg-[#2563eb] text-xl font-bold text-white">R</div>
            <div>
              <h1 className="text-xl font-semibold">RayIP 管理端</h1>
              <p className="mt-1 text-sm text-[#667085]">用户、钱包、库存、订单和 Runtime</p>
            </div>
          </div>
          <label className="block text-sm font-medium">管理员</label>
          <input className="field mt-2" value={username} onChange={(event) => setUsername(event.target.value)} />
          <label className="mt-4 block text-sm font-medium">密码</label>
          <input className="field mt-2" type="password" value={password} onChange={(event) => setPassword(event.target.value)} />
          <Button className="mt-6 w-full" type="submit" disabled={login.isPending}>登录</Button>
          {login.isError ? <p className="mt-3 text-sm text-[#c2410c]">{errorText(login.error)}</p> : null}
        </form>
      </div>
    );
  }

  return (
    <div className="min-h-screen bg-[#f3f4f6] text-[#1f2430]">
      <aside className="fixed inset-y-0 left-0 hidden w-[270px] border-r border-[#e5e7eb] bg-white lg:block">
        <div className="flex h-[72px] items-center gap-3 px-6">
          <div className="grid size-9 place-items-center rounded-md bg-[#2563eb] text-lg font-bold text-white">R</div>
          <div className="text-2xl font-semibold text-[#2563eb]">RayIP Admin</div>
        </div>
        <nav className="space-y-2 px-3">
          {([
            ["管理概览", Gauge, true],
            ["用户", Users, false],
            ["钱包和充值", WalletCards, false],
            ["产品和库存", Boxes, false],
            ["订单", ListChecks, false],
            ["家宽节点", Server, false],
            ["Runtime", Activity, false],
            ["风控和设置", Settings, false],
          ] satisfies Array<[string, LucideIcon, boolean]>).map(([label, Icon, active]) => (
            <button key={String(label)} className={`flex h-10 w-full items-center gap-3 rounded-md px-3 text-left text-sm ${active ? "bg-[#eef2f7] text-[#2563eb]" : "text-[#344054]"}`}>
              <Icon className="size-4" />
              <span>{label}</span>
            </button>
          ))}
        </nav>
      </aside>

      <div className="lg:pl-[270px]">
        <header className="sticky top-0 z-10 flex h-[60px] items-center justify-between border-b border-[#e5e7eb] bg-white px-6">
          <button className="grid size-9 place-items-center rounded-md hover:bg-[#eef2f7]" aria-label="菜单">
            <Menu className="size-5" />
          </button>
          <Button variant="outline" onClick={() => void queryClient.invalidateQueries()}>
            <RefreshCw className="size-4" />
            刷新
          </Button>
        </header>

        <main className="p-6">
          <div className="mb-6">
            <h1 className="text-2xl font-semibold">运营控制台</h1>
            <p className="mt-2 text-sm text-[#667085]">M2 商业闭环：用户、充值、可售库存、订单发货和 Runtime 状态</p>
          </div>

          <section className="grid gap-4 xl:grid-cols-5">
            <Metric label="用户" value={`${users.data?.total ?? 0}`} icon={<Users className="size-5" />} />
            <Metric label="支付单" value={`${payments.data?.total ?? 0}`} icon={<CreditCard className="size-5" />} />
            <Metric label="库存 IP" value={`${inventory.data?.total ?? 0}`} icon={<Boxes className="size-5" />} />
            <Metric label="订单" value={`${orders.data?.total ?? 0}`} icon={<ListChecks className="size-5" />} />
            <Metric label="节点" value={`${nodes.data?.total ?? 0}`} icon={<Server className="size-5" />} />
          </section>

          <section className="mt-5 grid gap-5 2xl:grid-cols-[1.1fr_0.9fr]">
            <Panel title="用户和充值订单">
              <div className="grid gap-5 xl:grid-cols-2">
                <DataTable
                  headers={["邮箱", "状态", "创建"]}
                  rows={(users.data?.items ?? []).map((item) => [item.email, item.status, dateText(item.created_at)])}
                />
                <DataTable
                  headers={["金额", "状态", "交易号"]}
                  rows={(payments.data?.items ?? []).map((item) => [money(item.amount_cents), item.status, item.provider_trade_no || "-"])}
                />
              </div>
            </Panel>

            <Panel title="钱包流水">
              <DataTable
                headers={["类型", "金额", "余额", "引用"]}
                rows={(ledger.data?.items ?? []).map((item) => [item.type, money(item.amount_cents), money(item.balance_after_cents), item.reference_id.slice(0, 8)])}
              />
            </Panel>
          </section>

          <section className="mt-5 grid gap-5 2xl:grid-cols-[0.9fr_1.1fr]">
            <Panel title="产品、价格和库存">
              <div className="mb-5 grid gap-3 md:grid-cols-3">
                {(products.data?.prices ?? []).map((price) => (
                  <div key={`${price.protocol}-${price.duration_days}`} className="rounded-lg border border-[#e5e7eb] p-3">
                    <div className="text-sm font-medium">{price.protocol} · {price.duration_days} 天</div>
                    <div className="mt-2 text-lg font-semibold">{money(price.unit_cents)}</div>
                  </div>
                ))}
              </div>
              <form
                className="grid gap-3 md:grid-cols-3"
                onSubmit={(event) => {
                  event.preventDefault();
                  addInventory.mutate();
                }}
              >
                <input className="field" value={inventoryForm.line_id} onChange={(event) => setInventoryForm({ ...inventoryForm, line_id: event.target.value })} placeholder="line_id" />
                <input className="field" value={inventoryForm.node_id} onChange={(event) => setInventoryForm({ ...inventoryForm, node_id: event.target.value })} placeholder="node_id" />
                <input className="field" value={inventoryForm.ip} onChange={(event) => setInventoryForm({ ...inventoryForm, ip: event.target.value })} placeholder="ip" />
                <input className="field" value={inventoryForm.port} onChange={(event) => setInventoryForm({ ...inventoryForm, port: event.target.value })} placeholder="port" />
                <input className="field" value={inventoryForm.protocols} onChange={(event) => setInventoryForm({ ...inventoryForm, protocols: event.target.value })} placeholder="SOCKS5,HTTP" />
                <Button type="submit" disabled={addInventory.isPending}>新增库存</Button>
              </form>
              {addInventory.isError ? <p className="mt-3 text-sm text-[#c2410c]">{errorText(addInventory.error)}</p> : null}
              <DataTable
                className="mt-5"
                headers={["IP", "节点", "协议", "状态"]}
                rows={(inventory.data?.items ?? []).map((item) => [`${item.ip}:${item.port}`, item.node_id, item.protocols.join("/"), item.status])}
              />
            </Panel>

            <Panel title="订单和代理生命周期">
              <AdminOrderTable orders={orders.data?.items ?? []} onRetry={(id) => retryOrder.mutate(id)} retrying={retryOrder.isPending} />
              <AdminProxyTable className="mt-5" proxies={proxies.data?.items ?? []} onReconcile={(id) => reconcileProxy.mutate(id)} reconciling={reconcileProxy.isPending} />
            </Panel>
          </section>

          <section className="mt-5">
            <Panel title="节点可售和 Runtime 状态">
              <DataTable
                headers={["节点", "状态", "能力", "最后在线"]}
                rows={(nodes.data?.items ?? []).map((item) => [item.code, item.status, item.capabilities?.join("/") || "-", dateText(item.last_online_at)])}
              />
              <div className="mt-4 flex items-center gap-2 rounded-lg border border-[#fde68a] bg-[#fffbeb] p-3 text-sm text-[#92400e]">
                <AlertTriangle className="size-4" />
                可售库存必须同时满足 node_runtime_status.sellable、线路启用、库存可用、无 hold、协议能力匹配。
              </div>
            </Panel>
          </section>
        </main>
      </div>
    </div>
  );
}

function Metric({ label, value, icon }: { label: string; value: string; icon: ReactNode }) {
  return (
    <div className="rounded-lg bg-white p-4 shadow-[0_1px_2px_rgba(15,23,42,0.05)]">
      <div className="flex items-center justify-between text-[#667085]">
        <span className="text-sm">{label}</span>
        {icon}
      </div>
      <div className="mt-3 text-2xl font-semibold">{value}</div>
    </div>
  );
}

function Panel({ title, children }: { title: string; children: ReactNode }) {
  return (
    <section className="rounded-lg bg-white p-5 shadow-[0_1px_2px_rgba(15,23,42,0.05)]">
      <div className="mb-4 flex items-center gap-2">
        <Shield className="size-4 text-[#2563eb]" />
        <h2 className="text-base font-semibold">{title}</h2>
      </div>
      {children}
    </section>
  );
}

function DataTable({ headers, rows, className = "" }: { headers: string[]; rows: Array<Array<ReactNode>>; className?: string }) {
  return (
    <div className={`overflow-x-auto ${className}`}>
      <table className="data-table">
        <thead>
          <tr>{headers.map((header) => <th key={header}>{header}</th>)}</tr>
        </thead>
        <tbody>
          {rows.length ? rows.map((row, index) => (
            <tr key={index}>{row.map((cell, cellIndex) => <td key={cellIndex}>{cell}</td>)}</tr>
          )) : (
            <tr><td colSpan={headers.length} className="text-center text-[#667085]">暂无数据</td></tr>
          )}
        </tbody>
      </table>
    </div>
  );
}

function AdminOrderTable({ orders, onRetry, retrying }: { orders: Order[]; onRetry: (id: string) => void; retrying: boolean }) {
  return (
    <div className="overflow-x-auto">
      <table className="data-table">
        <thead>
          <tr>
            <th>订单</th>
            <th>用户</th>
            <th>金额</th>
            <th>状态</th>
            <th>操作</th>
          </tr>
        </thead>
        <tbody>
          {orders.length ? orders.map((item) => (
            <tr key={item.id}>
              <td>{item.id.slice(0, 8)}</td>
              <td>{item.user_id.slice(0, 8)}</td>
              <td>{money(item.amount_cents)}</td>
              <td>{item.failure_reason ? `${item.status} · ${item.failure_reason}` : item.status}</td>
              <td>
                <Button size="sm" variant="outline" disabled={retrying || item.status !== "FULFILLMENT_FAILED"} onClick={() => onRetry(item.id)}>
                  重试发货
                </Button>
              </td>
            </tr>
          )) : (
            <tr><td colSpan={5} className="text-center text-[#667085]">暂无数据</td></tr>
          )}
        </tbody>
      </table>
    </div>
  );
}

function AdminProxyTable({ proxies, onReconcile, reconciling, className = "" }: { proxies: ProxyAccount[]; onReconcile: (id: string) => void; reconciling: boolean; className?: string }) {
  return (
    <div className={`overflow-x-auto ${className}`}>
      <table className="data-table">
        <thead>
          <tr>
            <th>代理</th>
            <th>节点</th>
            <th>IP</th>
            <th>状态</th>
            <th>生命周期</th>
            <th>操作</th>
          </tr>
        </thead>
        <tbody>
          {proxies.length ? proxies.map((item) => (
            <tr key={item.id}>
              <td>{item.id.slice(0, 8)}</td>
              <td>{item.node_id}</td>
              <td>{item.listen_ip}:{item.port}</td>
              <td>{item.status}</td>
              <td>{item.lifecycle_status}</td>
              <td>
                <Button size="sm" variant="outline" disabled={reconciling} onClick={() => onReconcile(item.id)}>
                  Runtime 对账
                </Button>
              </td>
            </tr>
          )) : (
            <tr><td colSpan={6} className="text-center text-[#667085]">暂无数据</td></tr>
          )}
        </tbody>
      </table>
    </div>
  );
}

function dateText(value?: string) {
  if (!value) return "-";
  return new Date(value).toLocaleString("zh-CN", { hour12: false });
}

function errorText(error: unknown) {
  return error instanceof Error ? error.message : "请求失败";
}
