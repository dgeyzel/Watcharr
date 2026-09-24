import { describe, expect, it } from "vitest";
import { TIERS, type Tier, type WatchedStatus } from "@/types";
import { tierSlot, UNRATED_TEXT, type TierSlotSize } from "./tierSlot";

const STATUSES: WatchedStatus[] = [
	"FINISHED",
	"WATCHING",
	"PLANNED",
	"HOLD",
	"DROPPED",
];
const SIZES: TierSlotSize[] = ["page", "poster"];
const TIER_VALUES: (Tier | null)[] = [...TIERS, null];

describe("tierSlot", () => {
	// Every status x {each letter, null} x {page, poster}.
	for (const status of STATUSES) {
		for (const tier of TIER_VALUES) {
			for (const size of SIZES) {
				it(`${status} / ${tier ?? "null"} / ${size}`, () => {
					const slot = tierSlot(status, tier, size);
					if (status === "PLANNED") {
						expect(slot).toEqual({ kind: "none" });
					} else if (tier) {
						expect(slot).toEqual({ kind: "tier", letter: tier });
					} else if (status === "FINISHED" || status === "WATCHING") {
						expect(slot).toEqual({
							kind: "unrated",
							text: UNRATED_TEXT[size],
						});
					} else {
						expect(slot).toEqual({ kind: "none" });
					}
				});
			}
		}
	}

	it("uses the exact unrated text from the spec", () => {
		expect(tierSlot("FINISHED", null, "page")).toEqual({
			kind: "unrated",
			text: "Watched but not yet rated",
		});
		expect(tierSlot("WATCHING", undefined, "poster")).toEqual({
			kind: "unrated",
			text: "not rated yet",
		});
	});

	it("planned never shows a saved tier", () => {
		expect(tierSlot("PLANNED", "S", "page")).toEqual({ kind: "none" });
	});

	it("no status (not on the list) has no slot", () => {
		expect(tierSlot(undefined, "A", "page")).toEqual({ kind: "none" });
	});
});
