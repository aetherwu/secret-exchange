"use client";

import Link from "next/link";
import { usePathname } from "next/navigation";

const tabs = [
  { href: "/discover", label: "Discover" },
  { href: "/feed", label: "Feed" },
  { href: "/contacts", label: "Contacts" },
  { href: "/me", label: "Me" },
];

export default function NavBar() {
  const pathname = usePathname();

  return (
    <nav className="fixed bottom-0 left-0 right-0 bg-white border-t border-gray-200 pb-[env(safe-area-inset-bottom)] z-50 md:top-0 md:bottom-auto md:border-t-0 md:border-b">
      <div className="max-w-2xl mx-auto flex justify-around">
        {tabs.map((tab) => {
          const active = pathname.startsWith(tab.href);
          return (
            <Link
              key={tab.href}
              href={tab.href}
              className={`flex-1 py-3 text-center text-sm font-medium transition-colors ${
                active
                  ? "text-blue-600 border-b-2 border-blue-600 md:border-b-0 md:border-t-2"
                  : "text-gray-500 hover:text-gray-700"
              }`}
            >
              {tab.label}
            </Link>
          );
        })}
      </div>
    </nav>
  );
}
