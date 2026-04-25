"use client";

import { Suspense } from "react";
import { useEffect, useState, useRef, useCallback } from "react";
import { useRouter, useSearchParams } from "next/navigation";
import { api } from "@/lib/api";
import { useRequireAuth } from "@/lib/useAuth";

const PRIVACY_OPTIONS = [
  { value: -1, label: "Public display" },
  { value: 0, label: "Public exchange" },
  { value: 1, label: "Friend-only" },
  { value: 2, label: "Invite-only" },
] as const;

const MAX_CHARS = 800;
const DRAFT_DEBOUNCE_MS = 1500;

function draftKey(questionId: string): string {
  return `draft_q_${questionId}`;
}

function WriteForm() {
  const ready = useRequireAuth();
  const router = useRouter();
  const searchParams = useSearchParams();

  const questionId = searchParams.get("questionId");
  const answerId = searchParams.get("answerId"); // present during exchange flow

  const [content, setContent] = useState("");
  const [privacy, setPrivacy] = useState(0);
  const [submitting, setSubmitting] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [draftRecovered, setDraftRecovered] = useState(false);
  const debounceRef = useRef<ReturnType<typeof setTimeout> | null>(null);

  // Recover draft from localStorage
  useEffect(() => {
    if (!questionId) return;
    const saved = localStorage.getItem(draftKey(questionId));
    if (saved && saved.trim().length > 0) {
      setContent(saved);
      setDraftRecovered(true);
    }
  }, [questionId]);

  // Auto-save draft with debounce
  const saveDraft = useCallback(
    (text: string) => {
      if (!questionId) return;
      if (debounceRef.current) clearTimeout(debounceRef.current);
      debounceRef.current = setTimeout(() => {
        localStorage.setItem(draftKey(questionId), text);
      }, DRAFT_DEBOUNCE_MS);
    },
    [questionId]
  );

  const handleContentChange = (val: string) => {
    if (val.length > MAX_CHARS) return;
    setContent(val);
    saveDraft(val);
    if (draftRecovered) setDraftRecovered(false);
  };

  const clearDraft = useCallback(() => {
    if (!questionId) return;
    localStorage.removeItem(draftKey(questionId));
  }, [questionId]);

  const handleSubmit = async () => {
    if (!content.trim() || !questionId) return;
    setSubmitting(true);
    setError(null);

    try {
      if (answerId) {
        // Exchange flow: write my answer and exchange with target answer
        const res = await api("/answer/exchange", {
          ID: answerId,
          Answer: content.trim(),
          Private: privacy,
        });
        if (res.errCode !== 0) {
          setError(res.errMsg || "Exchange failed");
          return;
        }
        clearDraft();
        router.push(`/answer/${answerId}`);
      } else {
        // Normal answer creation
        const res = await api<{ id?: string }>("/answer/create", {
          ID: questionId,
          Answer: content.trim(),
          Private: privacy,
        });
        if (res.errCode !== 0) {
          setError(res.errMsg || "Failed to submit answer");
          return;
        }
        clearDraft();
        const newId = res.id;
        router.push(`/answer/${newId}`);
      }
    } catch {
      setError("Network error. Please try again.");
    } finally {
      setSubmitting(false);
    }
  };

  if (!ready) return null;

  if (!questionId) {
    return (
      <div className="max-w-2xl mx-auto px-4 py-12 text-center">
        <p className="text-gray-500">No question specified.</p>
        <button
          onClick={() => router.push("/discover")}
          className="mt-4 text-blue-600 hover:underline"
        >
          Go to Discover
        </button>
      </div>
    );
  }

  return (
    <div className="max-w-2xl mx-auto px-4 py-6 flex flex-col min-h-[80vh]">
      <div className="flex items-center justify-between mb-4">
        <button
          onClick={() => router.back()}
          className="text-gray-500 hover:text-gray-700 text-sm"
        >
          &larr; Back
        </button>
        <span className="text-sm text-gray-400">
          {answerId ? "Write & Exchange" : "Write Answer"}
        </span>
      </div>

      {draftRecovered && (
        <div className="bg-yellow-50 border border-yellow-200 rounded-lg px-4 py-2 mb-4 flex items-center justify-between">
          <span className="text-sm text-yellow-800">
            Draft recovered from last session
          </span>
          <button
            onClick={() => {
              setContent("");
              clearDraft();
              setDraftRecovered(false);
            }}
            className="text-xs text-yellow-600 hover:underline ml-4"
          >
            Discard
          </button>
        </div>
      )}

      {/* Textarea */}
      <div className="flex-1 flex flex-col">
        <textarea
          className="flex-1 w-full bg-white rounded-xl shadow-sm p-6 text-base leading-relaxed resize-none focus:outline-none focus:ring-2 focus:ring-blue-500 min-h-[300px]"
          placeholder="Write your truth here..."
          value={content}
          onChange={(e) => handleContentChange(e.target.value)}
          maxLength={MAX_CHARS}
          autoFocus
        />
        <div className="flex items-center justify-between mt-2 px-1">
          <span
            className={`text-sm ${
              content.length >= MAX_CHARS ? "text-red-500" : "text-gray-400"
            }`}
          >
            {content.length}/{MAX_CHARS}
          </span>
        </div>
      </div>

      {/* Privacy selector */}
      <div className="mt-4 mb-4">
        <label className="block text-sm font-medium text-gray-600 mb-1">
          Privacy
        </label>
        <select
          value={privacy}
          onChange={(e) => setPrivacy(Number(e.target.value))}
          className="w-full bg-white border border-gray-300 rounded-lg py-2 px-3 text-sm focus:outline-none focus:ring-2 focus:ring-blue-500"
        >
          {PRIVACY_OPTIONS.map((opt) => (
            <option key={opt.value} value={opt.value}>
              {opt.label}
            </option>
          ))}
        </select>
      </div>

      {error && (
        <p className="text-sm text-red-500 mb-4 text-center">{error}</p>
      )}

      {/* Submit button */}
      <button
        onClick={handleSubmit}
        disabled={submitting || !content.trim()}
        className="w-full bg-blue-600 text-white rounded-lg py-3 px-6 hover:bg-blue-700 disabled:opacity-50 disabled:cursor-not-allowed transition-colors"
      >
        {submitting
          ? "Submitting..."
          : answerId
            ? "Submit & Exchange"
            : "Submit Answer"}
      </button>
    </div>
  );
}

export default function WritePage() {
  return (
    <Suspense
      fallback={
        <div className="flex items-center justify-center min-h-[60vh]">
          <div className="animate-pulse text-gray-400">Loading...</div>
        </div>
      }
    >
      <WriteForm />
    </Suspense>
  );
}
