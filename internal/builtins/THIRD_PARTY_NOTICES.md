# Third-party notices — built-in colormaps

The built-in palettes that ship with palettedb are well-known data-visualization
colormaps created by third parties. Their color data (the `data/*.txt` lookup
tables) is used under the licenses below. palettedb claims no ownership of this
data.

| Colormap | Author(s) | License | Source |
| --- | --- | --- | --- |
| viridis | Stéfan van der Walt, Nathaniel J. Smith, Eric Firing | CC0-1.0 | https://github.com/BIDS/colormap |
| plasma  | Stéfan van der Walt, Nathaniel J. Smith | CC0-1.0 | https://github.com/BIDS/colormap |
| inferno | Stéfan van der Walt, Nathaniel J. Smith | CC0-1.0 | https://github.com/BIDS/colormap |
| magma   | Stéfan van der Walt, Nathaniel J. Smith | CC0-1.0 | https://github.com/BIDS/colormap |
| cividis | Jamie R. Nuñez, Christopher R. Anderton, Ryan S. Renslow | CC0-1.0 | https://doi.org/10.1371/journal.pone.0199239 |
| turbo   | Google LLC (Anton Mikhailov) | Apache-2.0 | https://gist.github.com/mikhailov-work/ee72ba4191942acecc03fe6da94fc73f |
| mako    | Michael Waskom (seaborn) | BSD-3-Clause | https://github.com/mwaskom/seaborn |
| rocket  | Michael Waskom (seaborn) | BSD-3-Clause | https://github.com/mwaskom/seaborn |
| flare   | Michael Waskom (seaborn) | BSD-3-Clause | https://github.com/mwaskom/seaborn |
| crest   | Michael Waskom (seaborn) | BSD-3-Clause | https://github.com/mwaskom/seaborn |
| vlag    | Michael Waskom (seaborn) | BSD-3-Clause | https://github.com/mwaskom/seaborn |
| icefire | Michael Waskom (seaborn) | BSD-3-Clause | https://github.com/mwaskom/seaborn |

## License texts

- **CC0-1.0** (viridis, plasma, inferno, magma, cividis): public-domain dedication.
  Full text in [`LICENSE-CC0-1.0.txt`](./LICENSE-CC0-1.0.txt).
- **Apache-2.0** (turbo), © 2019 Google LLC. Full text in
  [`LICENSE-Apache-2.0.txt`](./LICENSE-Apache-2.0.txt).
- **BSD-3-Clause** (seaborn colormaps), © Michael L. Waskom. Full text in
  [`LICENSE-BSD-3-Clause.txt`](./LICENSE-BSD-3-Clause.txt).

## Notes

- The lookup tables were extracted verbatim (256 RGB samples each) from the
  canonical distributions (matplotlib / BIDS colormap, the turbo gist, and
  seaborn) and stored as `data/<name>.txt` (one `r g b` float triple per line).
- The data is unmodified; only the serialization format differs.
