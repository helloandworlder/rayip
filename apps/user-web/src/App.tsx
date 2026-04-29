import {
  BadgeCheck,
  Box,
  ChevronDown,
  CircleUserRound,
  CreditCard,
  Gauge,
  Gift,
  Home,
  KeyRound,
  Languages,
  Menu,
  MessageSquare,
  Moon,
  Network,
  ReceiptText,
  Share2,
  ShieldCheck,
  WalletCards,
} from "lucide-react";
import type { LucideIcon } from "lucide-react";
import { Button } from "@/components/ui/button";

type NavItem = {
  label: string;
  icon: LucideIcon;
  active?: boolean;
  children?: string[];
};

type NavGroup = {
  title: string;
  items: NavItem[];
};

const groups: NavGroup[] = [
  {
    title: "",
    items: [{ label: "概览", icon: Home, active: true }],
  },
  {
    title: "代理",
    items: [
      { label: "静态住宅代理", icon: Box, children: ["购买", "已购 IP 列表"] },
      { label: "动态住宅代理", icon: Network, children: ["购买流量包", "IP 提取", "账号管理"] },
    ],
  },
  {
    title: "工具",
    items: [
      { label: "开发者 App", icon: KeyRound },
      { label: "代理验证工具", icon: ShieldCheck },
    ],
  },
  {
    title: "账单",
    items: [
      { label: "账户充值", icon: WalletCards },
      { label: "计费", icon: ReceiptText },
      { label: "优惠券", icon: Gift },
      { label: "实名认证", icon: BadgeCheck },
    ],
  },
  {
    title: "推广",
    items: [{ label: "推广账户", icon: Share2 }],
  },
  {
    title: "帮助和支持",
    items: [{ label: "反馈建议", icon: MessageSquare }],
  },
];

const regions = [
  { name: "纽约", count: 5688 },
  { name: "洛杉矶", count: 13534 },
  { name: "芝加哥", count: 1786 },
  { name: "达拉斯", count: 941 },
];

export function App() {
  return (
    <div className="min-h-screen bg-[#f3f4f6] text-[#1f2430]">
      <aside className="fixed inset-y-0 left-0 hidden w-[292px] border-r border-[#e5e7eb] bg-white md:block">
        <div className="flex h-[76px] items-center gap-3 px-7">
          <div className="grid size-9 place-items-center rounded-md bg-[#3f73f6] text-lg font-bold text-white">
            R
          </div>
          <div className="text-3xl font-semibold tracking-normal text-[#3f73f6]">RayIP</div>
        </div>
        <nav className="space-y-5 px-3 py-3">
          {groups.map((group) => (
            <div key={group.title || "root"} className="space-y-2">
              {group.title ? (
                <div className="px-2 text-sm font-medium text-[#3f73f6]">{group.title}</div>
              ) : null}
              {group.items.map((item) => {
                const Icon = item.icon;
                return (
                  <div key={item.label}>
                    <button
                      className={[
                        "flex h-11 w-full items-center gap-3 rounded-md px-4 text-left text-[15px]",
                        item.active ? "bg-[#eef2f7] text-[#3f73f6]" : "text-[#202533]",
                      ].join(" ")}
                    >
                      <Icon className="size-5" />
                      <span className="flex-1">{item.label}</span>
                      {item.children ? <ChevronDown className="size-4" /> : null}
                    </button>
                    {item.children ? (
                      <div className="mt-1 space-y-1 pl-14 text-[15px] text-[#7b8494]">
                        {item.children.map((child) => (
                          <button key={child} className="block h-9">
                            {child}
                          </button>
                        ))}
                      </div>
                    ) : null}
                  </div>
                );
              })}
            </div>
          ))}
        </nav>
      </aside>

      <div className="md:pl-[292px]">
        <header className="sticky top-0 z-10 flex h-[64px] items-center justify-between border-b border-[#e5e7eb] bg-white px-6">
          <button className="grid size-9 place-items-center rounded-md hover:bg-[#eef2f7]" aria-label="折叠菜单">
            <Menu className="size-6" />
          </button>
          <div className="flex items-center gap-3 text-sm font-medium">
            <span>账户余额: ¥230.00</span>
            <Button size="sm">充值</Button>
            <button className="grid size-9 place-items-center rounded-md bg-[#f1f3f6]">
              <Moon className="size-5" />
            </button>
            <button className="hidden items-center gap-1 sm:flex">
              <Languages className="size-4" />
              中文
              <ChevronDown className="size-4" />
            </button>
            <button className="flex items-center gap-2">
              <span className="grid size-9 place-items-center rounded-full bg-[#eef0f3]">I</span>
              <ChevronDown className="size-4" />
            </button>
          </div>
        </header>

        <main className="p-6 md:p-8">
          <div className="mb-7 flex items-center justify-between">
            <div>
              <h1 className="text-2xl font-semibold">控制面板</h1>
              <p className="mt-2 text-sm text-[#6b7280]">静态家宽 IP 的购买与交付入口</p>
            </div>
            <Button variant="outline">
              <CircleUserRound className="size-4" />
              登录壳
            </Button>
          </div>

          <section className="grid gap-5 xl:grid-cols-2">
            <Panel title="静态住宅代理" subtitle="固定 IP 地址，稳定可靠">
              <div className="grid gap-3 sm:grid-cols-2">
                <Button size="lg">立即购买</Button>
                <Button variant="outline" size="lg">
                  查看已购
                </Button>
              </div>
            </Panel>
            <Panel title="账户状态" subtitle="余额、活跃代理与近期消耗">
              <div className="grid grid-cols-3 gap-3">
                <Metric label="余额" value="¥230.00" />
                <Metric label="活跃代理" value="0" />
                <Metric label="本月流量" value="0 GB" />
              </div>
            </Panel>
          </section>

          <section className="mt-5 rounded-lg bg-white p-6 shadow-[0_1px_2px_rgba(15,23,42,0.05)]">
            <div className="mb-5 flex items-start justify-between gap-4">
              <div>
                <h2 className="text-lg font-semibold">静态家宽购买</h2>
                <p className="mt-1 text-sm text-[#6b7280]">T1 阶段先固定购买入口和信息结构，真实库存会在 T6 接入。</p>
              </div>
              <Button variant="outline">
                <Gauge className="size-4" />
                可售闸门
              </Button>
            </div>
            <div className="grid gap-4 lg:grid-cols-[1fr_320px]">
              <div className="grid gap-4 sm:grid-cols-2 xl:grid-cols-4">
                <SelectBlock label="用途" value="电商 / 社媒" />
                <SelectBlock label="IP 类型" value="原生住宅" />
                <SelectBlock label="协议" value="SOCKS5 + HTTP" />
                <SelectBlock label="时长" value="30 天" />
              </div>
              <div className="rounded-lg border border-[#e5e7eb] p-4">
                <div className="text-sm text-[#6b7280]">订单预览</div>
                <div className="mt-2 text-2xl font-semibold">¥6.00</div>
                <Button className="mt-4 w-full">余额支付</Button>
              </div>
            </div>
            <div className="mt-5 grid gap-3 sm:grid-cols-2 xl:grid-cols-4">
              {regions.map((region) => (
                <button
                  key={region.name}
                  className="flex h-20 items-center justify-between rounded-lg border border-[#e5e7eb] bg-[#fbfcfe] px-4 text-left hover:border-[#9db7ff]"
                >
                  <span className="font-medium">{region.name}</span>
                  <span className="text-sm text-[#3f73f6]">{region.count} 可售</span>
                </button>
              ))}
            </div>
          </section>
        </main>
      </div>
    </div>
  );
}

function Panel({
  title,
  subtitle,
  children,
}: {
  title: string;
  subtitle: string;
  children: React.ReactNode;
}) {
  return (
    <section className="rounded-lg bg-white p-7 shadow-[0_1px_2px_rgba(15,23,42,0.05)]">
      <h2 className="text-lg font-semibold">{title}</h2>
      <p className="mt-2 text-sm text-[#6b7280]">{subtitle}</p>
      <div className="mt-8">{children}</div>
    </section>
  );
}

function Metric({ label, value }: { label: string; value: string }) {
  return (
    <div className="rounded-lg border border-[#e5e7eb] bg-[#fbfcfe] p-3">
      <div className="text-xs text-[#6b7280]">{label}</div>
      <div className="mt-1 font-semibold">{value}</div>
    </div>
  );
}

function SelectBlock({ label, value }: { label: string; value: string }) {
  return (
    <button className="h-[76px] rounded-lg border border-[#dfe4ef] bg-white px-4 text-left hover:border-[#9db7ff]">
      <div className="text-xs text-[#6b7280]">{label}</div>
      <div className="mt-2 font-medium">{value}</div>
    </button>
  );
}
