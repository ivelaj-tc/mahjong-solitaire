import type { ReactNode } from "react";

import type { Category } from "../../types";

interface CategoryScreenProps {
  themeToggle: ReactNode;
  categories: Category[];
  statusMessage: string;
  categoryChoices: Record<string, Category>;
  playerKey: string;
  categoryChoice: Category | null;
  myCategoryChoice?: Category;
  onSelect: (category: Category) => void;
}

export default function CategoryScreen({
  themeToggle,
  categories,
  statusMessage,
  categoryChoices,
  playerKey,
  categoryChoice,
  myCategoryChoice,
  onSelect,
}: CategoryScreenProps) {
  return (
    <div className="flex min-h-screen flex-col items-center justify-center gap-8 p-8">
      {themeToggle}
      <h1 className="font-display text-3xl font-bold text-[var(--jade)]">Choose Your Category</h1>
      <p className="text-lg text-[var(--muted)]">{statusMessage}</p>
      <div className="flex flex-wrap justify-center gap-3">
        {categories.map((category) => {
          const chosenBy = Object.entries(categoryChoices).find(([, value]) => value === category)?.[0];
          const isMine = chosenBy === playerKey;
          const isTaken = Boolean(chosenBy && !isMine);
          const isSelected = categoryChoice === category || isMine;
          return (
            <button
              key={category}
              onClick={() => onSelect(category)}
              disabled={isTaken || Boolean(myCategoryChoice)}
              className={`rounded-full border-2 px-5 py-2 text-sm font-semibold capitalize transition-all ${
                isSelected
                  ? "border-[var(--jade)] bg-[var(--jade)]/15 text-[var(--jade)]"
                  : "border-[var(--gold)] bg-[var(--surface-soft)] text-[var(--foreground)] hover:border-[var(--jade)]"
              } ${isTaken ? "opacity-50" : ""}`}
            >
              {category.replace("_", " ")}
            </button>
          );
        })}
      </div>
      {myCategoryChoice && (
        <p className="text-sm text-[var(--muted)]">You selected: {myCategoryChoice}</p>
      )}
    </div>
  );
}
