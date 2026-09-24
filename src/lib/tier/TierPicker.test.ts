import { fireEvent, render, screen } from "@testing-library/svelte";
import { describe, expect, it, vi } from "vitest";
import TierPicker from "./TierPicker.svelte";

describe("TierPicker", () => {
	it("has six tier buttons plus clear, with the current one pressed", () => {
		render(TierPicker, { tier: "B", onChange: () => {} });
		for (const g of ["S", "A", "B", "C", "D", "F"]) {
			const btn = screen.getByRole("button", { name: `Tier ${g}` });
			expect(btn).toHaveAttribute("aria-pressed", g === "B" ? "true" : "false");
		}
		expect(screen.getByRole("button", { name: "Clear tier" })).toBeEnabled();
	});

	it("calls onChange when a tier is clicked", async () => {
		const onChange = vi.fn();
		render(TierPicker, { tier: null, onChange });
		await fireEvent.click(screen.getByRole("button", { name: "Tier A" }));
		expect(onChange).toHaveBeenCalledWith("A");
	});

	it("does nothing when the current tier is clicked again", async () => {
		const onChange = vi.fn();
		render(TierPicker, { tier: "A", onChange });
		await fireEvent.click(screen.getByRole("button", { name: "Tier A" }));
		expect(onChange).not.toHaveBeenCalled();
	});

	it("clears the tier", async () => {
		const onChange = vi.fn();
		render(TierPicker, { tier: "C", onChange });
		await fireEvent.click(screen.getByRole("button", { name: "Clear tier" }));
		expect(onChange).toHaveBeenCalledWith(null);
	});

	it("clear is disabled when there is no tier", () => {
		render(TierPicker, { tier: null, onChange: () => {} });
		expect(screen.getByRole("button", { name: "Clear tier" })).toBeDisabled();
	});

	it("is keyboard operable: arrow keys move focus, Enter picks", async () => {
		const onChange = vi.fn();
		render(TierPicker, { tier: null, onChange });
		const s = screen.getByRole("button", { name: "Tier S" });
		s.focus();
		await fireEvent.keyDown(s, { key: "ArrowRight" });
		const a = screen.getByRole("button", { name: "Tier A" });
		expect(document.activeElement).toBe(a);
		await fireEvent.keyDown(a, { key: "ArrowLeft" });
		expect(document.activeElement).toBe(s);
		// Buttons are native, Enter/Space trigger a click.
		await fireEvent.click(document.activeElement as HTMLElement);
		expect(onChange).toHaveBeenCalledWith("S");
	});

	it("disabled blocks changes", async () => {
		const onChange = vi.fn();
		render(TierPicker, { tier: null, onChange, disabled: true });
		await fireEvent.click(screen.getByRole("button", { name: "Tier A" }));
		expect(onChange).not.toHaveBeenCalled();
	});
});
