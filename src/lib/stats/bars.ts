/** One bar of a BarChart. */
export interface Bar {
	label: string;
	value: number;
	/** Optional css class for the bar (e.g. a tier colour). */
	barClass?: string;
}
