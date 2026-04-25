"use client";

import { useEffect, useState, useCallback } from "react";
import { useRouter } from "next/navigation";
import { api } from "@/lib/api";
import { useRequireAuth } from "@/lib/useAuth";

interface Question {
  id: string;
  content: string;
  level: number;
  answerCount: number;
}

interface RandomResponse {
  errCode: number;
  question: Question | null;
  limit: number;
  answerCount: number;
  exchangedAnswerCount: number;
  exchangedUserCount: number;
}

export default function DiscoverPage() {
  const ready = useRequireAuth();
  const router = useRouter();
  const [question, setQuestion] = useState<Question | null>(null);
  const [limitRemaining, setLimitRemaining] = useState<number | null>(null);
  const [allAnswered, setAllAnswered] = useState(false);
  const [loading, setLoading] = useState(true);

  const fetchRandom = useCallback(async () => {
    setLoading(true);
    try {
      const res = await api<RandomResponse>("/question/random");
      if (res.errCode !== 0) return;

      if (!res.question) {
        setAllAnswered(true);
        setQuestion(null);
      } else {
        setAllAnswered(false);
        setQuestion(res.question);
      }
      setLimitRemaining(res.limit);
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    if (ready) fetchRandom();
  }, [ready, fetchRandom]);

  if (!ready) return null;

  if (loading) {
    return (
      <div className="flex items-center justify-center min-h-[60vh]">
        <div className="animate-pulse text-gray-400">Loading...</div>
      </div>
    );
  }

  // Daily limit reached
  if (limitRemaining === 0 && !allAnswered) {
    return (
      <div className="max-w-2xl mx-auto px-4 py-12">
        <div className="bg-white rounded-xl shadow-sm p-6 text-center">
          <div className="text-4xl mb-4">&#9203;</div>
          <h2 className="text-xl font-medium mb-2">
            You&apos;ve used all your answers for today
          </h2>
          <p className="text-gray-500">
            Come back tomorrow to discover more questions and exchange truths.
          </p>
        </div>
      </div>
    );
  }

  // All questions answered
  if (allAnswered) {
    return (
      <div className="max-w-2xl mx-auto px-4 py-12">
        <div className="bg-white rounded-xl shadow-sm p-6 text-center">
          <div className="text-4xl mb-4">&#127881;</div>
          <h2 className="text-xl font-medium mb-2">
            You&apos;ve answered every question!
          </h2>
          <p className="text-gray-500">
            Congratulations! Check back later for new questions.
          </p>
        </div>
      </div>
    );
  }

  return (
    <div className="max-w-2xl mx-auto px-4 py-8">
      {/* Rate limit indicator */}
      {limitRemaining !== null && (
        <p className="text-sm text-gray-500 text-center mb-6">
          {limitRemaining} answer{limitRemaining !== 1 ? "s" : ""} remaining
          today
        </p>
      )}

      {/* Question card */}
      {question && (
        <div className="bg-white rounded-xl shadow-sm p-6 flex flex-col items-center min-h-[40vh] justify-center">
          <p className="text-xs text-gray-400 uppercase tracking-wide mb-4">
            Level {question.level} &middot; {question.answerCount}{" "}
            {question.answerCount === 1 ? "answer" : "answers"}
          </p>
          <h1 className="text-xl font-medium text-center leading-relaxed mb-8 max-w-lg">
            {question.content}
          </h1>

          <div className="flex gap-4 w-full max-w-xs">
            <button
              onClick={fetchRandom}
              className="flex-1 py-3 px-6 rounded-lg border border-gray-300 text-gray-600 hover:bg-gray-50 transition-colors"
            >
              Skip
            </button>
            <button
              onClick={() =>
                router.push(`/write?questionId=${question.id}`)
              }
              className="flex-1 bg-blue-600 text-white rounded-lg py-3 px-6 hover:bg-blue-700 transition-colors"
            >
              Write my answer
            </button>
          </div>
        </div>
      )}
    </div>
  );
}
