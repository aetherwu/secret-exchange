"use client";

import { useRequireAuth } from "@/lib/useAuth";

export default function FeedPage() {
  const ready = useRequireAuth();
  if (!ready) return null;

  return (
    <div className="max-w-2xl mx-auto px-4 py-8">
      <h1 className="text-xl font-medium mb-4">Activity Feed</h1>
      <div className="bg-white rounded-xl shadow-sm p-6 text-center text-gray-400">
        Coming soon
      </div>
    </div>
  );
}
