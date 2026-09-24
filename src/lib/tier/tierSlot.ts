/**
 * The tier slot rule (FORK_SPEC D5), in one place. Components render from
 * this and never repeat the rule.
 *
 * - Finished or Watching with a tier: the letter.
 * - Finished or Watching without a tier: "Watched but not yet rated" (page)
 *   or "not rated yet" (poster).
 * - Planned: no tier slot at all, even if a tier is saved.
 * - On hold / dropped (admin only): the letter if tiered, otherwise nothing.
 */
import type { Tier, WatchedStatus } from "@/types";

export type TierSlotSize = "page" | "poster";

export type TierSlot =
	| { kind: "tier"; letter: Tier }
	| { kind: "unrated"; text: string }
	| { kind: "none" };

export const UNRATED_TEXT: Record<TierSlotSize, string> = {
	page: "Watched but not yet rated",
	poster: "not rated yet",
};

export function tierSlot(
	status: WatchedStatus | undefined,
	tier: Tier | null | undefined,
	size: TierSlotSize,
): TierSlot {
	if (!status || status === "PLANNED") {
		return { kind: "none" };
	}
	if (tier) {
		return { kind: "tier", letter: tier };
	}
	if (status === "FINISHED" || status === "WATCHING") {
		return { kind: "unrated", text: UNRATED_TEXT[size] };
	}
	return { kind: "none" };
}
