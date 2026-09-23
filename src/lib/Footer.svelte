<footer>
	<p>
		This site is a <a
			href="https://github.com/dgeyzel/Watcharr"
			target="_blank"
			rel="noopener noreferrer">fork</a
		>
		of
		<a href="https://watcharr.app/" target="_blank" rel="noopener noreferrer"
			>Watcharr</a
		>
	</p>
	<p class="tmdb">
		This product uses the TMDB API but is not endorsed or certified by TMDB.
	</p>
</footer>

<style lang="scss">
	footer {
		display: flex;
		flex-flow: column;
		align-items: center;
		gap: 4px;
		padding: 20px 16px;
		margin-top: 20px;
		text-align: center;
		font-size: 13px;
		color: var(--text-color-accent);
		overflow-wrap: anywhere;

		p {
			margin: 0;
		}

		a {
			color: var(--text-color);
			text-decoration: underline;

			&:focus-visible {
				outline: 2px solid var(--text-color);
				outline-offset: 2px;
				border-radius: 2px;
			}
		}

		.tmdb {
			font-size: 12px;
		}
	}
</style>
