"use client";

import { use, useEffect, useState, useCallback } from "react";
import { useRouter } from "next/navigation";
import { api, getOpenID } from "@/lib/api";
import { useRequireAuth } from "@/lib/useAuth";

interface Question {
  id: string;
  content: string;
  level: number;
}

interface User {
  openID: string;
  nickname?: string;
  avatar?: string;
}

interface Answer {
  id: string;
  questionID: string;
  content: string;
  length: number;
  privacy: number;
  exchangeCount: number;
  beExchangedCount: number;
  createdAt?: string;
  user?: User;
}

interface Comment {
  id: string;
  content: string;
  userOpenID: string;
  createdAt?: string;
}

interface ExchangeEntry {
  id: string;
  userOpenID: string;
  content: string;
  nickname?: string;
}

interface DetailResponse {
  errCode: number;
  errMsg?: string;
  answer: Answer;
  question: Question;
  owner: boolean;
  exchanged: boolean;
  exchangeds: ExchangeEntry[] | null;
  comments: Comment[] | null;
  myAnswer?: Answer | null;
  isFriend?: boolean;
}

const PRIVACY_LABELS: Record<number, string> = {
  [-1]: "Public display",
  0: "Public exchange",
  1: "Friend-only",
  2: "Invite-only",
};

function privacyBadgeColor(privacy: number): string {
  switch (privacy) {
    case -1:
      return "bg-green-100 text-green-700";
    case 0:
      return "bg-blue-100 text-blue-700";
    case 1:
      return "bg-yellow-100 text-yellow-700";
    case 2:
      return "bg-red-100 text-red-700";
    default:
      return "bg-gray-100 text-gray-700";
  }
}

export default function AnswerDetailPage({
  params,
}: {
  params: Promise<{ id: string }>;
}) {
  const { id } = use(params);
  const ready = useRequireAuth();
  const router = useRouter();

  const [data, setData] = useState<DetailResponse | null>(null);
  const [loading, setLoading] = useState(true);
  const [exchanging, setExchanging] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const myOpenID = getOpenID();

  const fetchDetail = useCallback(async () => {
    setLoading(true);
    try {
      const res = await api<DetailResponse>("/answer/detail", {
        ID: id,
      });
      if (res.errCode && res.errCode !== 0) {
        setError(res.errMsg || "Failed to load answer");
        return;
      }
      setData(res);
    } catch {
      setError("Network error");
    } finally {
      setLoading(false);
    }
  }, [id]);

  useEffect(() => {
    if (ready) fetchDetail();
  }, [ready, fetchDetail]);

  const handleExchangeWithExisting = async () => {
    if (!data?.myAnswer) return;
    setExchanging(true);
    setError(null);
    try {
      const res = await api("/answer/exchange-answer", {
        ID: id,
      });
      if (res.errCode && res.errCode !== 0) {
        setError(res.errMsg || "Exchange failed");
        return;
      }
      await fetchDetail();
    } catch {
      setError("Network error");
    } finally {
      setExchanging(false);
    }
  };

  if (!ready) return null;

  if (loading) {
    return (
      <div className="flex items-center justify-center min-h-[60vh]">
        <div className="animate-pulse text-gray-400">Loading...</div>
      </div>
    );
  }

  if (error && !data) {
    return (
      <div className="max-w-2xl mx-auto px-4 py-12 text-center">
        <p className="text-red-500 mb-4">{error}</p>
        <button
          onClick={() => router.push("/discover")}
          className="text-blue-600 hover:underline"
        >
          Go to Discover
        </button>
      </div>
    );
  }

  if (!data) return null;

  const { answer, question, owner: isOwner, exchanged: isExchanged, exchangeds, myAnswer, isFriend } =
    data;
  const user = answer.user || { openID: "", nickname: "Someone" };

  // ── Owner view ──
  if (isOwner) {
    return (
      <div className="max-w-2xl mx-auto px-4 py-6">
        <button
          onClick={() => router.back()}
          className="text-gray-500 hover:text-gray-700 text-sm mb-4 inline-block"
        >
          &larr; Back
        </button>

        {/* Question context */}
        <div className="bg-gray-100 rounded-lg px-4 py-3 mb-4">
          <p className="text-sm text-gray-500 mb-1">Question</p>
          <p className="text-base font-medium">{question.content}</p>
        </div>

        {/* My answer */}
        <div className="bg-white rounded-xl shadow-sm p-6 mb-4">
          <div className="flex items-center justify-between mb-3">
            <span className="text-sm text-gray-500">Your answer</span>
            <span
              className={`text-xs px-2 py-0.5 rounded-full ${privacyBadgeColor(answer.privacy)}`}
            >
              {PRIVACY_LABELS[answer.privacy] || "Unknown"}
            </span>
          </div>
          <p className="text-base leading-relaxed">{answer.content}</p>
        </div>

        {/* Exchanged users */}
        {exchangeds && exchangeds.length > 0 && (
          <div className="bg-white rounded-xl shadow-sm p-6">
            <h3 className="text-sm font-medium text-gray-500 mb-3">
              People who exchanged ({exchangeds.length})
            </h3>
            <ul className="space-y-3">
              {exchangeds.map((ex) => (
                <li key={ex.id} className="border-b border-gray-100 pb-3 last:border-0 last:pb-0">
                  <p className="text-sm font-medium text-gray-700 mb-1">
                    {ex.nickname || "Anonymous"}
                  </p>
                  <p className="text-base leading-relaxed">{ex.content}</p>
                </li>
              ))}
            </ul>
          </div>
        )}

        {error && (
          <p className="text-sm text-red-500 mt-4 text-center">{error}</p>
        )}
      </div>
    );
  }

  // ── Non-owner: Privacy=2 (invite-only) ──
  if (answer.privacy === 2) {
    return (
      <div className="max-w-2xl mx-auto px-4 py-12">
        <button
          onClick={() => router.back()}
          className="text-gray-500 hover:text-gray-700 text-sm mb-4 inline-block"
        >
          &larr; Back
        </button>
        <div className="bg-white rounded-xl shadow-sm p-6 text-center">
          <div className="text-3xl mb-3">&#128274;</div>
          <h2 className="text-xl font-medium mb-2">Invite-only</h2>
          <p className="text-gray-500">
            This answer is only visible through a direct invitation.
          </p>
        </div>
      </div>
    );
  }

  // ── Non-owner: Privacy=1, not friend ──
  if (answer.privacy === 1 && isFriend === false) {
    return (
      <div className="max-w-2xl mx-auto px-4 py-12">
        <button
          onClick={() => router.back()}
          className="text-gray-500 hover:text-gray-700 text-sm mb-4 inline-block"
        >
          &larr; Back
        </button>
        <div className="bg-white rounded-xl shadow-sm p-6 text-center">
          <div className="text-3xl mb-3">&#128101;</div>
          <h2 className="text-xl font-medium mb-2">Friend-only</h2>
          <p className="text-gray-500">
            Exchange requires friendship. Add this person as a friend first.
          </p>
        </div>
      </div>
    );
  }

  // ── Non-owner: already exchanged ──
  if (isExchanged) {
    // Find my exchange entry
    const myExchange = exchangeds?.find((ex) => ex.userOpenID === myOpenID);

    return (
      <div className="max-w-2xl mx-auto px-4 py-6">
        <button
          onClick={() => router.back()}
          className="text-gray-500 hover:text-gray-700 text-sm mb-4 inline-block"
        >
          &larr; Back
        </button>

        {/* Question context */}
        <div className="bg-gray-100 rounded-lg px-4 py-3 mb-4">
          <p className="text-sm text-gray-500 mb-1">Question</p>
          <p className="text-base font-medium">{question.content}</p>
        </div>

        {/* Their answer */}
        <div className="bg-white rounded-xl shadow-sm p-6 mb-4">
          <p className="text-sm text-gray-500 mb-2">
            {user.nickname || "Their"}&apos;s answer
          </p>
          <p className="text-base leading-relaxed">{answer.content}</p>
        </div>

        {/* My exchanged answer */}
        {myExchange && (
          <div className="bg-blue-50 rounded-xl shadow-sm p-6">
            <p className="text-sm text-blue-600 mb-2">Your exchanged answer</p>
            <p className="text-base leading-relaxed">{myExchange.content}</p>
          </div>
        )}

        {error && (
          <p className="text-sm text-red-500 mt-4 text-center">{error}</p>
        )}
      </div>
    );
  }

  // ── Non-owner, not exchanged, has prior answer ──
  if (myAnswer) {
    return (
      <div className="max-w-2xl mx-auto px-4 py-6">
        <button
          onClick={() => router.back()}
          className="text-gray-500 hover:text-gray-700 text-sm mb-4 inline-block"
        >
          &larr; Back
        </button>

        {/* Question context */}
        <div className="bg-gray-100 rounded-lg px-4 py-3 mb-4">
          <p className="text-sm text-gray-500 mb-1">Question</p>
          <p className="text-base font-medium">{question.content}</p>
        </div>

        {/* Hint about the answer */}
        <div className="bg-white rounded-xl shadow-sm p-6 mb-4 text-center">
          <p className="text-gray-500 mb-1">
            {user.nickname || "Someone"} answered{" "}
            <span className="font-medium text-gray-700">
              {answer.length} characters
            </span>
          </p>
          <p className="text-sm text-gray-400">
            Exchange your answer to reveal theirs
          </p>
        </div>

        {error && (
          <p className="text-sm text-red-500 mb-4 text-center">{error}</p>
        )}

        <button
          onClick={handleExchangeWithExisting}
          disabled={exchanging}
          className="w-full bg-blue-600 text-white rounded-lg py-3 px-6 hover:bg-blue-700 disabled:opacity-50 disabled:cursor-not-allowed transition-colors"
        >
          {exchanging ? "Exchanging..." : "Use your answer to exchange"}
        </button>
      </div>
    );
  }

  // ── Non-owner, not exchanged, no prior answer ──
  return (
    <div className="max-w-2xl mx-auto px-4 py-6">
      <button
        onClick={() => router.back()}
        className="text-gray-500 hover:text-gray-700 text-sm mb-4 inline-block"
      >
        &larr; Back
      </button>

      {/* Question context */}
      <div className="bg-gray-100 rounded-lg px-4 py-3 mb-4">
        <p className="text-sm text-gray-500 mb-1">Question</p>
        <p className="text-base font-medium">{question.content}</p>
      </div>

      {/* Hint about the answer */}
      <div className="bg-white rounded-xl shadow-sm p-6 mb-4 text-center">
        <p className="text-gray-500 mb-1">
          {user.nickname || "Someone"} answered{" "}
          <span className="font-medium text-gray-700">
            {answer.length} characters
          </span>
        </p>
        <p className="text-sm text-gray-400">
          Write your own answer to exchange and reveal theirs
        </p>
      </div>

      <button
        onClick={() =>
          router.push(
            `/write?questionId=${question.id}&answerId=${answer.id}`
          )
        }
        className="w-full bg-blue-600 text-white rounded-lg py-3 px-6 hover:bg-blue-700 transition-colors"
      >
        Write and exchange
      </button>
    </div>
  );
}
