import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import {
  Activity,
  AlertTriangle,
  Boxes,
  CircleDot,
  Copy,
  Gauge,
  ListChecks,
  Menu,
  PlayCircle,
  RefreshCw,
  Server,
  Settings,
  Shield,
  SlidersHorizontal,
  TerminalSquare,
  Trash2,
  Users,
  WalletCards,
} from "lucide-react";
import type { FormEvent, MouseEvent, ReactNode } from "react";
import { useMemo, useState } from "react";
import { Button } from "@/components/ui/button";

type NodeStatus = "ONLINE" | "OFFLINE";
type Protocol = "SOCKS5" | "HTTP";
type ApplyStatus = "SUCCESS" | "FAILED" | "SKIPPED" | "DUPLICATE";

type NodeSummary = {
  id: string;
  code: string;
  status: NodeStatus;
  last_online_at: string;
  bundle_version: string;
  agent_version: string;
  xray_version: string;
  api_instance_id: string;
  session_id: string;
  capabilities: string[] | null;
  lease_expires_at?: string;
};

type NodeListResponse = {
  items: NodeSummary[];
  total: number;
};

type RuntimeAccount = {
  proxy_account_id: string;
  node_id: string;
  runtime_email: string;
  protocol: Protocol;
  listen_ip: string;
  port: number;
  username: string;
  password: string;
  egress_limit_bps: number;
  ingress_limit_bps: number;
  max_connections: number;
  status: "ENABLED" | "DISABLED" | "DELETED";
  policy_version: number;
  desired_generation: number;
  applied_generation: number;
  created_at: string;
  updated_at: string;
};

type RuntimeUsage = {
  rx_bytes: number;
  tx_bytes: number;
  active_connections: number;
  rx_bytes_per_second: number;
  tx_bytes_per_second: number;
};

type RuntimeDigest = {
  account_count: number;
  enabled_count: number;
  disabled_count: number;
  max_generation: number;
  hash: string;
};

type RuntimeResult = {
  command_id: string;
  proxy_account_id?: string;
  node_id?: string;
  operation?: string;
  status: ApplyStatus;
  error_code?: string;
  error_message?: string;
  applied_generation: number;
  usage?: RuntimeUsage;
  digest?: RuntimeDigest;
  created_at?: string;
};

type AccountListResponse = {
  items: RuntimeAccount[];
  total: number;
};

type ResultListResponse = {
  items: RuntimeResult[];
  total: number;
};

type RuntimeActionResponse = {
  account?: RuntimeAccount;
  result: RuntimeResult;
};

const nav = [
  { label: "管理概览", icon: Gauge },
  { label: "用户", icon: Users },
  { label: "钱包和充值", icon: WalletCards },
  { label: "产品和库存", icon: Boxes },
  { label: "家宽节点", icon: Server },
  { label: "Runtime Lab", icon: Activity, active: true },
  { label: "任务控制台", icon: ListChecks },
  { label: "Web SSH", icon: TerminalSquare },
  { label: "审计和设置", icon: Settings },
];

const baseURL = import.meta.env.VITE_API_BASE_URL ?? "";

async function apiJSON<T>(path: string, init?: RequestInit): Promise<T> {
  const response = await fetch(`${baseURL}${path}`, {
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

function fetchNodes(): Promise<NodeListResponse> {
  return apiJSON<NodeListResponse>("/api/admin/nodes");
}

function fetchAccounts(): Promise<AccountListResponse> {
  return apiJSON<AccountListResponse>("/api/admin/runtime-lab/accounts");
}

function fetchResults(accountID: string): Promise<ResultListResponse> {
  if (!accountID) return Promise.resolve({ items: [], total: 0 });
  return apiJSON<ResultListResponse>(`/api/admin/runtime-lab/accounts/${accountID}/results`);
}

export function App() {
  const queryClient = useQueryClient();
  const [selectedAccountID, setSelectedAccountID] = useState("");
  const [lastResult, setLastResult] = useState<RuntimeResult | null>(null);
  const [form, setForm] = useState({
    node_id: "",
    protocol: "SOCKS5" as Protocol,
    listen_ip: "127.0.0.1",
    port: 18080,
    username: "lab-user",
    password: "lab-pass",
    egress_limit_bps: 0,
    ingress_limit_bps: 0,
    max_connections: 2,
  });

  const nodes = useQuery({
    queryKey: ["admin-nodes"],
    queryFn: fetchNodes,
    refetchInterval: 5000,
  });
  const accounts = useQuery({
    queryKey: ["runtime-lab-accounts"],
    queryFn: fetchAccounts,
    refetchInterval: 5000,
  });
  const results = useQuery({
    queryKey: ["runtime-lab-results", selectedAccountID],
    queryFn: () => fetchResults(selectedAccountID),
    enabled: Boolean(selectedAccountID),
  });

  const onlineNodes = nodes.data?.items.filter((node) => node.status === "ONLINE") ?? [];
  const selectedNodeID = form.node_id || onlineNodes[0]?.id || "";
  const selectedAccount = accounts.data?.items.find((account) => account.proxy_account_id === selectedAccountID);
  const digest = lastResult?.digest;

  const invalidateLab = () => {
    void queryClient.invalidateQueries({ queryKey: ["runtime-lab-accounts"] });
    void queryClient.invalidateQueries({ queryKey: ["runtime-lab-results"] });
  };

  const createAccount = useMutation({
    mutationFn: () =>
      apiJSON<RuntimeActionResponse>("/api/admin/runtime-lab/accounts", {
        method: "POST",
        body: JSON.stringify({
          ...form,
          node_id: selectedNodeID,
          desired_generation: 1,
        }),
      }),
    onSuccess: (data) => {
      setLastResult(data.result);
      if (data.account) setSelectedAccountID(data.account.proxy_account_id);
      invalidateLab();
    },
  });

  const updatePolicy = useMutation({
    mutationFn: (account: RuntimeAccount) =>
      apiJSON<RuntimeActionResponse>(`/api/admin/runtime-lab/accounts/${account.proxy_account_id}/policy`, {
        method: "PATCH",
        body: JSON.stringify({
          egress_limit_bps: form.egress_limit_bps,
          ingress_limit_bps: form.ingress_limit_bps,
          max_connections: form.max_connections,
          desired_generation: account.desired_generation + 1,
        }),
      }),
    onSuccess: (data) => {
      setLastResult(data.result);
      invalidateLab();
    },
  });

  const runAction = useMutation({
    mutationFn: ({ account, action }: { account: RuntimeAccount; action: "disable" | "delete" | "usage" | "probe" }) => {
      const method = action === "usage" ? "GET" : action === "delete" ? "DELETE" : "POST";
      const suffix = action === "delete" ? "" : `/${action === "usage" ? "usage" : action}`;
      return apiJSON<RuntimeActionResponse>(`/api/admin/runtime-lab/accounts/${account.proxy_account_id}${suffix}`, { method });
    },
    onSuccess: (data) => {
      setLastResult(data.result);
      invalidateLab();
    },
  });

  const getDigest = useMutation({
    mutationFn: (nodeID: string) => apiJSON<RuntimeActionResponse>(`/api/admin/runtime-lab/nodes/${nodeID}/digest`),
    onSuccess: (data) => setLastResult(data.result),
  });

  const busy = createAccount.isPending || updatePolicy.isPending || runAction.isPending || getDigest.isPending;
  const visibleResults = useMemo(() => {
    if (lastResult) return [lastResult, ...(results.data?.items ?? []).filter((item) => item.command_id !== lastResult.command_id)];
    return results.data?.items ?? [];
  }, [lastResult, results.data?.items]);

  return (
    <div className="min-h-screen bg-[#f3f4f6] text-[#1f2430]">
      <aside className="fixed inset-y-0 left-0 hidden w-[260px] border-r border-[#e5e7eb] bg-white lg:block">
        <div className="flex h-[64px] items-center gap-3 px-5">
          <div className="grid size-8 place-items-center rounded-md bg-[#2563eb] text-sm font-bold text-white">R</div>
          <div>
            <div className="text-xl font-semibold text-[#2563eb]">RayIP</div>
            <div className="text-xs text-[#6b7280]">Admin</div>
          </div>
        </div>
        <nav className="space-y-1 px-3 py-4">
          {nav.map((item) => {
            const Icon = item.icon;
            return (
              <button
                key={item.label}
                className={[
                  "flex h-10 w-full items-center gap-3 rounded-md px-3 text-sm",
                  item.active ? "bg-[#eef2ff] text-[#2563eb]" : "text-[#394150] hover:bg-[#f6f8fb]",
                ].join(" ")}
              >
                <Icon className="size-4" />
                <span>{item.label}</span>
              </button>
            );
          })}
        </nav>
      </aside>

      <div className="lg:pl-[260px]">
        <header className="sticky top-0 z-10 flex h-[64px] items-center justify-between border-b border-[#e5e7eb] bg-white px-5">
          <div className="flex items-center gap-3">
            <button className="grid size-9 place-items-center rounded-md hover:bg-[#eef2f7]" aria-label="折叠菜单">
              <Menu className="size-5" />
            </button>
            <div>
              <div className="text-lg font-semibold">Runtime Lab</div>
              <div className="text-xs text-[#6b7280]">T2 直连在线 NodeAgent，验证账号增量控制</div>
            </div>
          </div>
          <div className="flex items-center gap-2">
            <Button variant="outline" onClick={() => void nodes.refetch()} disabled={busy}>
              <RefreshCw className="size-4" />
              刷新
            </Button>
            <Button variant="outline" onClick={() => selectedNodeID && getDigest.mutate(selectedNodeID)} disabled={!selectedNodeID || busy}>
              <Shield className="size-4" />
              Digest
            </Button>
          </div>
        </header>

        <main className="p-5">
          <section className="grid gap-4 md:grid-cols-4">
            <Stat title="在线节点" value={`${onlineNodes.length}`} hint={`总计 ${nodes.data?.total ?? 0} 台`} />
            <Stat title="Lab 账号" value={`${accounts.data?.total ?? 0}`} hint="仅管理端实验账号" />
            <Stat title="Digest 账号" value={`${digest?.account_count ?? "-"}`} hint={`水位 ${digest?.max_generation ?? "-"}`} />
            <Stat title="最近结果" value={lastResult?.status ?? "-"} hint={lastResult?.error_message || lastResult?.operation || "等待操作"} />
          </section>

          <section className="mt-4 grid gap-4 xl:grid-cols-[380px_1fr]">
            <form className="rounded-lg bg-white shadow-[0_1px_2px_rgba(15,23,42,0.05)]" onSubmit={(event) => submitCreate(event, createAccount.mutate)}>
              <PanelHead title="创建测试账号" hint="SOCKS5 / HTTP，按 proxy_account_id 固定 runtime email" />
              <div className="grid gap-3 p-5">
                <Label text="在线节点">
                  <select className="field" value={selectedNodeID} onChange={(event) => setForm({ ...form, node_id: event.target.value })}>
                    {onlineNodes.length ? (
                      onlineNodes.map((node) => (
                        <option key={node.id} value={node.id}>
                          {node.code}
                        </option>
                      ))
                    ) : (
                      <option value="">暂无在线节点</option>
                    )}
                  </select>
                </Label>
                <div className="grid grid-cols-2 gap-3">
                  <Label text="协议">
                    <select className="field" value={form.protocol} onChange={(event) => setForm({ ...form, protocol: event.target.value as Protocol })}>
                      <option value="SOCKS5">SOCKS5</option>
                      <option value="HTTP">HTTP</option>
                    </select>
                  </Label>
                  <Label text="端口">
                    <input className="field" type="number" value={form.port} onChange={(event) => setNumber("port", event.target.value, setForm)} />
                  </Label>
                </div>
                <div className="grid grid-cols-2 gap-3">
                  <Label text="用户名">
                    <input className="field" value={form.username} onChange={(event) => setForm({ ...form, username: event.target.value })} />
                  </Label>
                  <Label text="密码">
                    <input className="field" value={form.password} onChange={(event) => setForm({ ...form, password: event.target.value })} />
                  </Label>
                </div>
                <div className="grid grid-cols-3 gap-3">
                  <Label text="出站 B/s">
                    <input className="field" type="number" value={form.egress_limit_bps} onChange={(event) => setNumber("egress_limit_bps", event.target.value, setForm)} />
                  </Label>
                  <Label text="入站 B/s">
                    <input className="field" type="number" value={form.ingress_limit_bps} onChange={(event) => setNumber("ingress_limit_bps", event.target.value, setForm)} />
                  </Label>
                  <Label text="连接数">
                    <input className="field" type="number" value={form.max_connections} onChange={(event) => setNumber("max_connections", event.target.value, setForm)} />
                  </Label>
                </div>
                <Button type="submit" disabled={!selectedNodeID || busy}>
                  <PlayCircle className="size-4" />
                  创建并下发
                </Button>
              </div>
            </form>

            <section className="rounded-lg bg-white shadow-[0_1px_2px_rgba(15,23,42,0.05)]">
              <PanelHead title="Runtime Lab 账号" hint="重复 generation 不会再次改 Runtime，新 generation 覆盖策略" />
              <div className="overflow-x-auto">
                <table className="min-w-full text-left text-sm">
                  <thead className="bg-[#f8fafc] text-xs text-[#6b7280]">
                    <tr>
                      <th className="px-4 py-3 font-medium">账号</th>
                      <th className="px-4 py-3 font-medium">协议</th>
                      <th className="px-4 py-3 font-medium">连接</th>
                      <th className="px-4 py-3 font-medium">策略</th>
                      <th className="px-4 py-3 font-medium">状态</th>
                      <th className="px-4 py-3 font-medium">Generation</th>
                      <th className="px-4 py-3 font-medium">操作</th>
                    </tr>
                  </thead>
                  <tbody className="divide-y divide-[#edf0f4]">
                    {accounts.isLoading ? (
                      <EmptyRow colSpan={7} text="正在读取 Runtime Lab 账号..." />
                    ) : accounts.data?.items.length ? (
                      accounts.data.items.map((account) => (
                        <AccountRow
                          key={account.proxy_account_id}
                          account={account}
                          selected={account.proxy_account_id === selectedAccountID}
                          onSelect={() => setSelectedAccountID(account.proxy_account_id)}
                          onCopy={() => void navigator.clipboard?.writeText(connectionText(account))}
                          onPolicy={() => updatePolicy.mutate(account)}
                          onAction={(action) => runAction.mutate({ account, action })}
                          busy={busy}
                        />
                      ))
                    ) : (
                      <EmptyRow colSpan={7} text="暂无实验账号。选择在线节点后创建 SOCKS5 或 HTTP 账号。" />
                    )}
                  </tbody>
                </table>
              </div>
            </section>
          </section>

          <section className="mt-4 grid gap-4 xl:grid-cols-[1fr_360px]">
            <section className="rounded-lg bg-white shadow-[0_1px_2px_rgba(15,23,42,0.05)]">
              <PanelHead title="Apply Result" hint={selectedAccount ? selectedAccount.proxy_account_id : "选择账号后查看最近结果"} />
              <div className="overflow-x-auto">
                <table className="min-w-full text-left text-sm">
                  <thead className="bg-[#f8fafc] text-xs text-[#6b7280]">
                    <tr>
                      <th className="px-4 py-3 font-medium">状态</th>
                      <th className="px-4 py-3 font-medium">操作</th>
                      <th className="px-4 py-3 font-medium">Generation</th>
                      <th className="px-4 py-3 font-medium">Usage</th>
                      <th className="px-4 py-3 font-medium">错误</th>
                    </tr>
                  </thead>
                  <tbody className="divide-y divide-[#edf0f4]">
                    {visibleResults.length ? (
                      visibleResults.slice(0, 8).map((result) => <ResultRow key={result.command_id} result={result} />)
                    ) : (
                      <EmptyRow colSpan={5} text="暂无 apply 结果。" />
                    )}
                  </tbody>
                </table>
              </div>
            </section>

            <section className="rounded-lg bg-white shadow-[0_1px_2px_rgba(15,23,42,0.05)]">
              <PanelHead title="Digest" hint="账号数、generation 水位、hash" />
              <div className="space-y-3 p-5 text-sm">
                <KV label="账号数" value={`${digest?.account_count ?? "-"}`} />
                <KV label="启用 / 禁用" value={`${digest?.enabled_count ?? "-"} / ${digest?.disabled_count ?? "-"}`} />
                <KV label="最大 generation" value={`${digest?.max_generation ?? "-"}`} />
                <div>
                  <div className="mb-1 text-xs text-[#6b7280]">Hash</div>
                  <div className="break-all rounded-md bg-[#f8fafc] px-3 py-2 font-mono text-xs text-[#4b5565]">{digest?.hash || "-"}</div>
                </div>
              </div>
            </section>
          </section>

          {(nodes.isError || accounts.isError || createAccount.isError || updatePolicy.isError || runAction.isError || getDigest.isError) && (
            <div className="mt-4 flex items-center gap-3 rounded-lg border border-[#fed7aa] bg-[#fff7ed] px-4 py-3 text-sm text-[#b45309]">
              <AlertTriangle className="size-5" />
              {errorText(nodes.error || accounts.error || createAccount.error || updatePolicy.error || runAction.error || getDigest.error)}
            </div>
          )}
        </main>
      </div>
    </div>
  );
}

function submitCreate(event: FormEvent, mutate: () => void) {
  event.preventDefault();
  mutate();
}

function setNumber(
  key: "port" | "egress_limit_bps" | "ingress_limit_bps" | "max_connections",
  value: string,
  setForm: (updater: (current: AppForm) => AppForm) => void,
) {
  setForm((current) => ({ ...current, [key]: Number(value) || 0 }));
}

type AppForm = {
  node_id: string;
  protocol: Protocol;
  listen_ip: string;
  port: number;
  username: string;
  password: string;
  egress_limit_bps: number;
  ingress_limit_bps: number;
  max_connections: number;
};

function Stat({ title, value, hint }: { title: string; value: string; hint: string }) {
  return (
    <div className="rounded-lg bg-white p-5 shadow-[0_1px_2px_rgba(15,23,42,0.05)]">
      <div className="text-sm text-[#6b7280]">{title}</div>
      <div className="mt-2 text-2xl font-semibold">{value}</div>
      <div className="mt-1 truncate text-xs text-[#8a94a6]">{hint}</div>
    </div>
  );
}

function PanelHead({ title, hint }: { title: string; hint: string }) {
  return (
    <div className="flex min-h-[64px] items-center justify-between border-b border-[#e5e7eb] px-5 py-4">
      <div>
        <h2 className="font-semibold">{title}</h2>
        <p className="mt-1 text-sm text-[#6b7280]">{hint}</p>
      </div>
    </div>
  );
}

function Label({ text, children }: { text: string; children: ReactNode }) {
  return (
    <label className="grid gap-1 text-xs font-medium text-[#6b7280]">
      {text}
      {children}
    </label>
  );
}

function AccountRow({
  account,
  selected,
  onSelect,
  onCopy,
  onPolicy,
  onAction,
  busy,
}: {
  account: RuntimeAccount;
  selected: boolean;
  onSelect: () => void;
  onCopy: () => void;
  onPolicy: () => void;
  onAction: (action: "disable" | "delete" | "usage" | "probe") => void;
  busy: boolean;
}) {
  return (
    <tr className={selected ? "bg-[#f8fbff]" : ""} onClick={onSelect}>
      <td className="px-4 py-4">
        <div className="max-w-[180px] truncate font-medium">{account.proxy_account_id}</div>
        <div className="text-xs text-[#6b7280]">{account.username}</div>
      </td>
      <td className="px-4 py-4 text-[#4b5565]">{account.protocol}</td>
      <td className="px-4 py-4 text-[#4b5565]">
        {account.listen_ip}:{account.port}
      </td>
      <td className="px-4 py-4 text-xs text-[#4b5565]">
        <div>出 {formatBytes(account.egress_limit_bps)}/s</div>
        <div>入 {formatBytes(account.ingress_limit_bps)}/s</div>
        <div>连接 {account.max_connections || "不限"}</div>
      </td>
      <td className="px-4 py-4">
        <StatusBadge status={account.status} />
      </td>
      <td className="px-4 py-4 text-[#4b5565]">
        {account.applied_generation}/{account.desired_generation}
      </td>
      <td className="px-4 py-4">
        <div className="flex flex-wrap items-center gap-2">
          <Button variant="ghost" size="sm" onClick={stop(onCopy)} disabled={busy} title="复制连接">
            <Copy className="size-4" />
          </Button>
          <Button variant="ghost" size="sm" onClick={stop(onPolicy)} disabled={busy} title="更新策略">
            <SlidersHorizontal className="size-4" />
          </Button>
          <Button variant="outline" size="sm" onClick={stop(() => onAction("probe"))} disabled={busy}>
            测试
          </Button>
          <Button variant="outline" size="sm" onClick={stop(() => onAction("usage"))} disabled={busy}>
            Usage
          </Button>
          <Button variant="ghost" size="sm" onClick={stop(() => onAction("disable"))} disabled={busy}>
            禁用
          </Button>
          <Button variant="ghost" size="sm" onClick={stop(() => onAction("delete"))} disabled={busy} title="删除">
            <Trash2 className="size-4" />
          </Button>
        </div>
      </td>
    </tr>
  );
}

function ResultRow({ result }: { result: RuntimeResult }) {
  return (
    <tr>
      <td className="px-4 py-4">
        <ApplyBadge status={result.status} />
      </td>
      <td className="px-4 py-4 text-[#4b5565]">{result.operation || "-"}</td>
      <td className="px-4 py-4 text-[#4b5565]">{result.applied_generation}</td>
      <td className="px-4 py-4 text-xs text-[#4b5565]">
        <div>RX {formatBytes(result.usage?.rx_bytes ?? 0)}</div>
        <div>TX {formatBytes(result.usage?.tx_bytes ?? 0)}</div>
        <div>连接 {result.usage?.active_connections ?? 0}</div>
      </td>
      <td className="max-w-[320px] px-4 py-4 text-xs text-[#b45309]">{result.error_message || result.error_code || "-"}</td>
    </tr>
  );
}

function StatusBadge({ status }: { status: RuntimeAccount["status"] }) {
  return (
    <span
      className={[
        "inline-flex items-center gap-1 rounded-md px-2 py-1 text-xs font-medium",
        status === "ENABLED" ? "bg-[#e8f7ee] text-[#15803d]" : "bg-[#f1f3f6] text-[#64748b]",
      ].join(" ")}
    >
      <CircleDot className="size-3" />
      {status}
    </span>
  );
}

function ApplyBadge({ status }: { status: ApplyStatus }) {
  const style = status === "FAILED" ? "bg-[#fff7ed] text-[#b45309]" : status === "DUPLICATE" ? "bg-[#eef2ff] text-[#2563eb]" : "bg-[#e8f7ee] text-[#15803d]";
  return <span className={`inline-flex rounded-md px-2 py-1 text-xs font-medium ${style}`}>{status}</span>;
}

function EmptyRow({ colSpan, text }: { colSpan: number; text: string }) {
  return (
    <tr>
      <td className="px-5 py-8 text-[#6b7280]" colSpan={colSpan}>
        {text}
      </td>
    </tr>
  );
}

function KV({ label, value }: { label: string; value: string }) {
  return (
    <div className="flex items-center justify-between border-b border-[#edf0f4] pb-2">
      <span className="text-[#6b7280]">{label}</span>
      <span className="font-medium">{value}</span>
    </div>
  );
}

function connectionText(account: RuntimeAccount) {
  return `${account.protocol.toLowerCase()}://${account.username}:${account.password}@${account.listen_ip}:${account.port}`;
}

function formatBytes(value: number) {
  if (!value) return "不限";
  if (value >= 1024 * 1024) return `${(value / 1024 / 1024).toFixed(1)} MB`;
  if (value >= 1024) return `${(value / 1024).toFixed(1)} KB`;
  return `${value} B`;
}

function stop(fn: () => void) {
  return (event: MouseEvent) => {
    event.stopPropagation();
    fn();
  };
}

function errorText(error: unknown) {
  return error instanceof Error ? error.message : "Runtime Lab 操作失败";
}
