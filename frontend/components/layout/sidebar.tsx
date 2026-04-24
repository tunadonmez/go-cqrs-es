"use client";

import Link from "next/link";
import { usePathname } from "next/navigation";
import { clsx } from "clsx";

const links = [
  { href: "/dashboard", label: "Dashboard" },
  { href: "/wallets", label: "Wallets" },
  { href: "/ledger", label: "Ledger" },
  { href: "/commands", label: "Commands" },
  { href: "/dead-letters", label: "Dead Letters" },
  { href: "/operations", label: "Operations" }
];

export function Sidebar() {
  const pathname = usePathname();

  return (
    <aside className="w-full border-b border-slate-200/70 bg-white/70 px-4 py-4 backdrop-blur md:min-h-screen md:w-72 md:border-b-0 md:border-r">
      <div className="mb-6 flex items-center justify-between md:block">
        <div>
          <p className="text-xs font-semibold uppercase tracking-[0.3em] text-teal-700">
            CQRS Console
          </p>
          <h1 className="mt-2 text-2xl font-semibold text-slate-950">Wallet Admin</h1>
          <p className="mt-1 text-sm text-slate-500">
            Command and query flows kept separate on purpose.
          </p>
        </div>
      </div>
      <nav className="flex flex-wrap gap-2 md:flex-col">
        {links.map((link) => {
          const active = pathname === link.href || pathname.startsWith(`${link.href}/`);
          return (
            <Link
              key={link.href}
              href={link.href}
              className={clsx(
                "rounded-2xl px-4 py-3 text-sm font-medium transition",
                active
                  ? "bg-slate-950 text-white"
                  : "bg-slate-100/80 text-slate-700 hover:bg-slate-200"
              )}
            >
              {link.label}
            </Link>
          );
        })}
      </nav>
    </aside>
  );
}
