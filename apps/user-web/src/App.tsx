import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import {
  BadgeCheck,
  Box,
  ChevronDown,
  Copy,
  CreditCard,
  DatabaseZap,
  Gauge,
  Globe2,
  Home,
  KeyRound,
  Languages,
  ListChecks,
  Menu,
  Moon,
  ReceiptText,
  ShieldCheck,
  WalletCards,
} from "lucide-react";
import type { FormEvent, ReactNode } from "react";
import { useMemo, useState } from "react";
import { Button } from "@/components/ui/button";

type User = { id: string; email: string; status: string };
type Wallet = { balance_cents: number; held_cents: number; available_cents: number };
type ProductPrice = { product_id: string; protocol: Protocol; duration_days: number; unit_cents: number };
type Protocol = "SOCKS5" | "HTTP";
type CatalogLine = {
  line: { id: string; name: string; node_id: string; enabled: boolean };
  available: number;
  inventory_ids?: string[];
  sellable: boolean;
  reasons?: string[];
};
type CatalogRegion = {
  region: { id: string; name: string; country: string };
  cities: Array<{ city: { id: string; name: string }; lines: CatalogLine[]; available: number }>;
  available: number;
  disabled_reasons?: string[];
};
type Catalog = {
  product: { id: string; name: string; ip_type: string };
  prices: ProductPrice[];
  regions: CatalogRegion[];
  total_available: number;
};
type Order = {
  id: string;
  proxy_account_id: string;
  protocol: Protocol;
  duration_days: number;
  amount_cents: number;
  status: string;
  failure_reason?: string;
  created_at: string;
  expires_at?: string;
};
type ProxyAccount = {
  id: string;
  order_id: string;
  protocol: Protocol;
  listen_ip: string;
  port: number;
  username: string;
  password?: string;
  connection_uri?: string;
  lifecycle_status: string;
  expires_at: string;
};
type Ledger = {
  id: string;
  type: string;
  amount_cents: number;
  balance_after_cents: number;
  held_after_cents: number;
  reference_id: string;
  created_at: string;
};

const baseURL = import.meta.env.VITE_API_BASE_URL ?? "";

async function apiJSON<T>(path: string, init?: RequestInit): Promise<T> {
  const response = await fetch(`${baseURL}${path}`, {
    credentials: "include",
    ...init,
    headers: {
      "Content-Type": "application/json",
      ...init?.headers,
    },
  });
  if (!response.ok) {
    const text = await response.text();
    throw new Error(text || `请求失败: ${response.status}`);
  }
  return response.json() as Promise<T>;
}

const money = (cents: number) => `¥${(cents / 100).toFixed(2)}`;
const daysLeft = (expiresAt?: string) => {
  if (!expiresAt) return "-";
  const diff = new Date(expiresAt).getTime() - Date.now();
  return `${Math.max(0, Math.ceil(diff / 86400000))} 天`;
};

export function App() {
  const queryClient = useQueryClient();
  const [authMode, setAuthMode] = useState<"login" | "register">("login");
  const [email, setEmail] = useState("buyer@example.com");
  const [password, setPassword] = useState("secret123");
  const [rechargeYuan, setRechargeYuan] = useState(100);
  const [selectedProtocol, setSelectedProtocol] = useState<Protocol>("SOCKS5");
  const [durationDays, setDurationDays] = useState(30);
  const [selectedInventoryID, setSelectedInventoryID] = useState("");

  const me = useQuery({
    queryKey: ["me"],
    queryFn: () => apiJSON<{ user: User; wallet: Wallet }>("/api/me"),
    retry: false,
  });
  const catalog = useQuery({
    queryKey: ["catalog"],
    queryFn: () => apiJSON<{ catalog: Catalog }>("/api/catalog/static-residential"),
  });
  const orders = useQuery({
    queryKey: ["orders"],
    queryFn: () => apiJSON<{ items: Order[]; total: number }>("/api/orders"),
    enabled: me.isSuccess,
  });
  const proxies = useQuery({
    queryKey: ["proxies"],
    queryFn: () => apiJSON<{ items: ProxyAccount[]; total: number }>("/api/proxies"),
    enabled: me.isSuccess,
  });
  const ledger = useQuery({
    queryKey: ["ledger"],
    queryFn: () => apiJSON<{ items: Ledger[]; total: number }>("/api/admin/wallet-ledger"),
    enabled: false,
  });

  const auth = useMutation({
    mutationFn: async () => {
      if (authMode === "register") {
        await apiJSON("/api/auth/register", { method: "POST", body: JSON.stringify({ email, password }) });
      }
      return apiJSON<{ user: User; wallet: Wallet }>("/api/auth/login", {
        method: "POST",
        body: JSON.stringify({ email, password }),
      });
    },
    onSuccess: () => void queryClient.invalidateQueries(),
  });

  const recharge = useMutation({
    mutationFn: async () => {
      const amount = Math.round(rechargeYuan * 100);
      const created = await apiJSON<{ order: { id: string } }>("/api/payments/orders", {
        method: "POST",
        body: JSON.stringify({ amount_cents: amount }),
      });
      return apiJSON("/api/payments/mock-callback", {
        method: "POST",
        body: JSON.stringify({
          payment_order_id: created.order.id,
          provider_trade_no: `mock-${created.order.id}`,
          paid_amount_cents: amount,
        }),
      });
    },
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ["me"] });
    },
  });

  const selectedPrice = catalog.data?.catalog.prices.find(
    (price) => price.protocol === selectedProtocol && price.duration_days === durationDays,
  );
  const availableLines = useMemo(() => {
    const result: Array<{ label: string; line: CatalogLine; region: string; city: string }> = [];
    for (const region of catalog.data?.catalog.regions ?? []) {
      for (const city of region.cities) {
        for (const line of city.lines) {
          result.push({ label: `${region.region.name} / ${city.city.name} / ${line.line.name}`, line, region: region.region.name, city: city.city.name });
        }
      }
    }
    return result;
  }, [catalog.data]);

  const createOrder = useMutation({
    mutationFn: () =>
      apiJSON("/api/orders", {
        method: "POST",
        headers: { "Idempotency-Key": crypto.randomUUID() },
        body: JSON.stringify({
          product_id: "static-residential",
          inventory_id: selectedInventoryID,
          protocol: selectedProtocol,
          duration_days: durationDays,
          quantity: 1,
        }),
      }),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ["me"] });
      void queryClient.invalidateQueries({ queryKey: ["orders"] });
      void queryClient.invalidateQueries({ queryKey: ["proxies"] });
      void queryClient.invalidateQueries({ queryKey: ["catalog"] });
    },
  });

  const renewProxy = useMutation({
    mutationFn: (proxyID: string) =>
      apiJSON(`/api/proxies/${proxyID}/renew`, {
        method: "POST",
        headers: { "Idempotency-Key": crypto.randomUUID() },
        body: JSON.stringify({ duration_days: 30 }),
      }),
    onSuccess: () => void queryClient.invalidateQueries({ queryKey: ["proxies"] }),
  });

  const disableProxy = useMutation({
    mutationFn: (proxyID: string) => apiJSON(`/api/proxies/${proxyID}/disable`, { method: "POST" }),
    onSuccess: () => void queryClient.invalidateQueries({ queryKey: ["proxies"] }),
  });

  if (me.isError) {
    return (
      <div className="min-h-screen bg-[#f3f4f6] text-[#1f2430]">
        <div className="mx-auto grid min-h-screen max-w-[1120px] items-center px-6 lg:grid-cols-[1.1fr_420px]">
          <section>
            <div className="mb-8 flex items-center gap-3">
              <div className="grid size-11 place-items-center rounded-md bg-[#3f73f6] text-2xl font-bold text-white">R</div>
              <div className="text-4xl font-semibold text-[#3f73f6]">RayIP</div>
            </div>
            <h1 className="max-w-[680px] text-[34px] font-semibold leading-tight">静态住宅代理控制台</h1>
            <p className="mt-4 max-w-[620px] text-base text-[#667085]">注册、充值、查看真实可售线路，并在 Runtime 确认后复制代理凭据。</p>
          </section>
          <form
            className="rounded-lg bg-white p-7 shadow-[0_12px_36px_rgba(15,23,42,0.08)]"
            onSubmit={(event) => {
              event.preventDefault();
              auth.mutate();
            }}
          >
            <div className="mb-6 flex rounded-md bg-[#eef2f7] p-1">
              {(["login", "register"] as const).map((mode) => (
                <button
                  key={mode}
                  type="button"
                  onClick={() => setAuthMode(mode)}
                  className={`h-9 flex-1 rounded-md text-sm font-medium ${authMode === mode ? "bg-white text-[#3f73f6] shadow-sm" : "text-[#667085]"}`}
                >
                  {mode === "login" ? "登录" : "注册"}
                </button>
              ))}
            </div>
            <label className="block text-sm font-medium">邮箱</label>
            <input className="field mt-2" value={email} onChange={(event) => setEmail(event.target.value)} />
            <label className="mt-4 block text-sm font-medium">密码</label>
            <input className="field mt-2" type="password" value={password} onChange={(event) => setPassword(event.target.value)} />
            <Button className="mt-6 w-full" type="submit" disabled={auth.isPending}>
              {authMode === "login" ? "登录 RayIP" : "注册并登录"}
            </Button>
            {auth.isError ? <p className="mt-3 text-sm text-[#c2410c]">{errorText(auth.error)}</p> : null}
          </form>
        </div>
      </div>
    );
  }

  const wallet = me.data?.wallet ?? { balance_cents: 0, held_cents: 0, available_cents: 0 };

  return (
    <div className="min-h-screen bg-[#f3f4f6] text-[#1f2430]">
      <aside className="fixed inset-y-0 left-0 hidden w-[292px] border-r border-[#e5e7eb] bg-white md:block">
        <div className="flex h-[76px] items-center gap-3 px-7">
          <div className="grid size-9 place-items-center rounded-md bg-[#3f73f6] text-lg font-bold text-white">R</div>
          <div className="text-3xl font-semibold text-[#3f73f6]">RayIP</div>
        </div>
        <nav className="space-y-5 px-3 py-3">
          <NavGroup title="" items={[["概览", Home, true]]} />
          <NavGroup title="代理" items={[["静态住宅代理", Box, true], ["已购 IP 列表", DatabaseZap, false], ["代理验证工具", ShieldCheck, false]]} />
          <NavGroup title="账单" items={[["账户充值", WalletCards, false], ["计费记录", ReceiptText, false], ["实名认证", BadgeCheck, false]]} />
          <NavGroup title="工具" items={[["开发者 App", KeyRound, false], ["任务状态", ListChecks, false]]} />
        </nav>
      </aside>

      <div className="md:pl-[292px]">
        <header className="sticky top-0 z-10 flex h-[64px] items-center justify-between border-b border-[#e5e7eb] bg-white px-6">
          <button className="grid size-9 place-items-center rounded-md hover:bg-[#eef2f7]" aria-label="折叠菜单">
            <Menu className="size-6" />
          </button>
          <div className="flex items-center gap-3 text-sm font-medium">
            <span>账户余额: {money(wallet.available_cents)}</span>
            <Button size="sm" onClick={() => recharge.mutate()} disabled={recharge.isPending}>
              充值
            </Button>
            <button className="grid size-9 place-items-center rounded-md bg-[#f1f3f6]" aria-label="主题">
              <Moon className="size-5" />
            </button>
            <button className="hidden items-center gap-1 sm:flex">
              <Languages className="size-4" />
              中文
              <ChevronDown className="size-4" />
            </button>
            <span className="grid size-9 place-items-center rounded-full bg-[#eef0f3]">{me.data?.user.email.slice(0, 1).toUpperCase()}</span>
          </div>
        </header>

        <main className="p-6 md:p-8">
          <div className="mb-7 flex flex-wrap items-end justify-between gap-4">
            <div>
              <h1 className="text-2xl font-semibold">控制面板</h1>
              <p className="mt-2 text-sm text-[#6b7280]">静态家宽 IP 的购买、发货和生命周期入口</p>
            </div>
            <div className="flex items-center gap-2">
              <input className="field w-[120px]" type="number" value={rechargeYuan} onChange={(event) => setRechargeYuan(Number(event.target.value))} />
              <Button variant="outline" onClick={() => recharge.mutate()} disabled={recharge.isPending}>
                <CreditCard className="size-4" />
                模拟充值
              </Button>
            </div>
          </div>

          <section className="grid gap-5 xl:grid-cols-4">
            <Metric label="可用余额" value={money(wallet.available_cents)} />
            <Metric label="冻结金额" value={money(wallet.held_cents)} />
            <Metric label="真实可售库存" value={`${catalog.data?.catalog.total_available ?? 0}`} />
            <Metric label="已交付代理" value={`${proxies.data?.total ?? 0}`} />
          </section>

          <section className="mt-5 grid gap-5 xl:grid-cols-[1fr_340px]">
            <Panel title="静态住宅代理购买">
              <div className="grid gap-4 lg:grid-cols-4">
                <SelectBlock label="用途" value="电商 / 社媒 / 注册" />
                <SelectBlock label="IP 类型" value={catalog.data?.catalog.product.ip_type ?? "原生住宅"} />
                <label className="select-panel">
                  <span>协议</span>
                  <select value={selectedProtocol} onChange={(event) => setSelectedProtocol(event.target.value as Protocol)}>
                    <option value="SOCKS5">SOCKS5</option>
                    <option value="HTTP">HTTP</option>
                  </select>
                </label>
                <label className="select-panel">
                  <span>时长</span>
                  <select value={durationDays} onChange={(event) => setDurationDays(Number(event.target.value))}>
                    {[30, 60, 90, 180].map((days) => (
                      <option key={days} value={days}>
                        {days} 天
                      </option>
                    ))}
                  </select>
                </label>
              </div>

              <div className="mt-5 grid gap-3 lg:grid-cols-2 xl:grid-cols-3">
                {availableLines.map((item) => (
                  <button
                    key={item.line.line.id}
                    className={`line-card ${item.line.inventory_ids?.includes(selectedInventoryID) ? "selected" : ""}`}
                    onClick={() => item.line.sellable && setSelectedInventoryID(item.line.inventory_ids?.[0] ?? "")}
                    disabled={!item.line.sellable}
                  >
                    <span className="font-medium">{item.label}</span>
                    <span className={item.line.sellable ? "text-[#2563eb]" : "text-[#a16207]"}>
                      {item.line.sellable ? `${item.line.available} 可售` : (item.line.reasons ?? ["不可售"]).join(" / ")}
                    </span>
                  </button>
                ))}
              </div>
            </Panel>

            <Panel title="订单摘要">
              <div className="space-y-3 text-sm">
                <SummaryRow label="协议" value={selectedProtocol} />
                <SummaryRow label="时长" value={`${durationDays} 天`} />
                <SummaryRow label="数量" value="1" />
                <SummaryRow label="单价" value={money(selectedPrice?.unit_cents ?? 0)} />
                <div className="border-t border-[#e5e7eb] pt-4">
                  <SummaryRow label="应付" value={money(selectedPrice?.unit_cents ?? 0)} strong />
                </div>
              </div>
              <Button className="mt-5 w-full" disabled={!selectedInventoryID || createOrder.isPending} onClick={() => createOrder.mutate()}>
                余额支付
              </Button>
              {createOrder.isError ? <p className="mt-3 text-sm text-[#c2410c]">{errorText(createOrder.error)}</p> : null}
            </Panel>
          </section>

          <section className="mt-5 grid gap-5 xl:grid-cols-2">
            <Panel title="已购 IP 列表">
              <ProxyTable proxies={proxies.data?.items ?? []} onRenew={(id) => renewProxy.mutate(id)} onDisable={(id) => disableProxy.mutate(id)} />
            </Panel>
            <Panel title="订单和发货状态">
              <OrderTable orders={orders.data?.items ?? []} />
            </Panel>
          </section>
        </main>
      </div>
    </div>
  );
}

function NavGroup({ title, items }: { title: string; items: Array<[string, typeof Home, boolean]> }) {
  return (
    <div className="space-y-2">
      {title ? <div className="px-2 text-sm font-medium text-[#3f73f6]">{title}</div> : null}
      {items.map(([label, Icon, active]) => (
        <button key={label} className={`flex h-11 w-full items-center gap-3 rounded-md px-4 text-left text-[15px] ${active ? "bg-[#eef2f7] text-[#3f73f6]" : "text-[#202533]"}`}>
          <Icon className="size-5" />
          <span className="flex-1">{label}</span>
        </button>
      ))}
    </div>
  );
}

function Panel({ title, children }: { title: string; children: ReactNode }) {
  return (
    <section className="rounded-lg bg-white p-6 shadow-[0_1px_2px_rgba(15,23,42,0.05)]">
      <h2 className="mb-5 text-lg font-semibold">{title}</h2>
      {children}
    </section>
  );
}

function Metric({ label, value }: { label: string; value: string }) {
  return (
    <div className="rounded-lg bg-white p-5 shadow-[0_1px_2px_rgba(15,23,42,0.05)]">
      <div className="text-sm text-[#6b7280]">{label}</div>
      <div className="mt-2 text-2xl font-semibold">{value}</div>
    </div>
  );
}

function SelectBlock({ label, value }: { label: string; value: string }) {
  return (
    <div className="h-[76px] rounded-lg border border-[#dfe4ef] bg-white px-4 py-3">
      <div className="text-xs text-[#6b7280]">{label}</div>
      <div className="mt-2 font-medium">{value}</div>
    </div>
  );
}

function SummaryRow({ label, value, strong }: { label: string; value: string; strong?: boolean }) {
  return (
    <div className="flex items-center justify-between">
      <span className="text-[#6b7280]">{label}</span>
      <span className={strong ? "text-2xl font-semibold text-[#1f2430]" : "font-medium"}>{value}</span>
    </div>
  );
}

function ProxyTable({ proxies, onRenew, onDisable }: { proxies: ProxyAccount[]; onRenew: (id: string) => void; onDisable: (id: string) => void }) {
  if (!proxies.length) return <Empty label="Runtime ACK 后会显示代理凭据" />;
  return (
    <div className="overflow-x-auto">
      <table className="data-table">
        <thead>
          <tr>
            <th>IP:Port</th>
            <th>账号</th>
            <th>密码</th>
            <th>剩余</th>
            <th>操作</th>
          </tr>
        </thead>
        <tbody>
          {proxies.map((proxy) => (
            <tr key={proxy.id}>
              <td>{proxy.listen_ip}:{proxy.port}</td>
              <td>{proxy.username}</td>
              <td>{proxy.password || "-"}</td>
              <td>{daysLeft(proxy.expires_at)}</td>
              <td className="space-x-2 whitespace-nowrap">
                <Button size="sm" variant="outline" onClick={() => proxy.connection_uri && void navigator.clipboard.writeText(proxy.connection_uri)}>
                  <Copy className="size-3.5" />
                  复制
                </Button>
                <Button size="sm" variant="outline" onClick={() => onRenew(proxy.id)}>续费</Button>
                <Button size="sm" variant="outline" onClick={() => onDisable(proxy.id)}>停用</Button>
              </td>
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  );
}

function OrderTable({ orders }: { orders: Order[] }) {
  if (!orders.length) return <Empty label="暂无订单" />;
  return (
    <div className="overflow-x-auto">
      <table className="data-table">
        <thead>
          <tr>
            <th>订单号</th>
            <th>协议</th>
            <th>金额</th>
            <th>状态</th>
          </tr>
        </thead>
        <tbody>
          {orders.map((order) => (
            <tr key={order.id}>
              <td className="font-mono text-xs">{order.id.slice(0, 8)}</td>
              <td>{order.protocol}</td>
              <td>{money(order.amount_cents)}</td>
              <td>{order.status}{order.failure_reason ? ` · ${order.failure_reason}` : ""}</td>
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  );
}

function Empty({ label }: { label: string }) {
  return <div className="rounded-lg border border-dashed border-[#d8dde8] p-8 text-center text-sm text-[#6b7280]">{label}</div>;
}

function errorText(error: unknown) {
  return error instanceof Error ? error.message : "请求失败";
}
