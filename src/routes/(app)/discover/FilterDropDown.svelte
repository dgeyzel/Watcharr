<script lang="ts">
	import tooltip from "@/lib/actions/tooltip";
	import DropDown from "@/lib/DropDown.svelte";
	import {
		DiscoverFilter,
		SearchType,
		type DiscoverFilterOption,
		type DropDownItem,
	} from "@/types";

	interface Props {
		active: DiscoverFilter | undefined;
		discoverType: SearchType | undefined;
		onChange: () => void;
	}

	let { active = $bindable(), discoverType, onChange }: Props = $props();

	const dropDownOptions: { [x in DiscoverFilterOption]: DropDownItem } = {
		trending: {
			id: DiscoverFilter.trending,
			value: "Trending",
		},
		popular: {
			id: DiscoverFilter.popular,
			value: "Popular",
		},
		upcoming: {
			id: DiscoverFilter.upcoming,
			value: "Upcoming",
		},
		intheatres: {
			id: DiscoverFilter.inTheatres,
			value: "In Theatres",
		},
		streaming: {
			id: DiscoverFilter.streaming,
			value: "Streaming",
		},
		// advanced: {
		// 	id: "advanced",
		// 	value: "Advanced Search",
		// },
	};

	let options = $derived.by(() => {
		let o: DropDownItem[] = [dropDownOptions.trending];
		switch (discoverType) {
			case SearchType.movie:
				o.push(
					dropDownOptions.popular,
					dropDownOptions.upcoming,
					dropDownOptions.intheatres,
				);
				break;
			case SearchType.show:
				o.push(dropDownOptions.popular, dropDownOptions.upcoming);
				break;
			case SearchType.person:
				o.push(dropDownOptions.popular);
				break;
		}
		// o.push(dropDownOptions.advanced);
		return o;
	});
	let onMultiDiscover = $derived(discoverType === SearchType.multi);
</script>

<div
	use:tooltip={{
		text: "Must select a type first.",
		pos: "left",
		condition: onMultiDiscover,
	}}
>
	<DropDown
		placeholder="Trending"
		{options}
		isDropDownItem={true}
		bind:active
		{onChange}
		disabled={onMultiDiscover}
	/>
</div>
