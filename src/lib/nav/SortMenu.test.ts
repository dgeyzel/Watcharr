import { fireEvent, render, screen } from "@testing-library/svelte";
import { afterEach, describe, expect, it } from "vitest";
import SortMenu from "./SortMenu.svelte";
import { defaultSort, store } from "@/store.svelte";
import { TIERS, UserPermission, type PrivateUser } from "@/types";

afterEach(() => {
	store.activeSort = defaultSort;
	store.userInfo = undefined;
});

describe("SortMenu", () => {
	it("tiers are ordered S to F", () => {
		expect(TIERS).toEqual(["S", "A", "B", "C", "D", "F"]);
	});

	it("tier sort starts S first (down), then F first (up), then off", async () => {
		render(SortMenu);
		const tier = screen.getByRole("button", { name: "Tier" });
		await fireEvent.click(tier);
		expect(store.activeSort).toEqual(["TIER", "DOWN"]);
		await fireEvent.click(tier);
		expect(store.activeSort).toEqual(["TIER", "UP"]);
		await fireEvent.click(tier);
		expect(store.activeSort).toEqual([]);
	});

	it("other sorts start ascending", async () => {
		render(SortMenu);
		await fireEvent.click(screen.getByRole("button", { name: "Alphabetical" }));
		expect(store.activeSort).toEqual(["ALPHA", "UP"]);
	});

	it("visitors don't get the sorts that use private data", () => {
		render(SortMenu);
		expect(screen.queryByRole("button", { name: "Rating" })).toBeNull();
		expect(screen.queryByRole("button", { name: "Last Changed" })).toBeNull();
		expect(screen.getByRole("button", { name: "Tier" })).toBeInTheDocument();
	});

	it("the admin gets them", () => {
		store.userInfo = {
			id: 1,
			username: "admin",
			permissions: UserPermission.PERM_ADMIN,
		} as unknown as PrivateUser;
		render(SortMenu);
		expect(screen.getByRole("button", { name: "Rating" })).toBeInTheDocument();
	});
});
