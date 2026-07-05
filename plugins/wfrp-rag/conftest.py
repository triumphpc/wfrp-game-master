"""Conftest — makes the wfrp_rag plugin modules importable for tests."""

import importlib.util
import os
import sys

collect_ignore = ["__init__.py"]

PLUGIN_DIR = os.path.dirname(__file__)
sys.path.insert(0, PLUGIN_DIR)


def _load_plugin_module():
    init_path = os.path.join(PLUGIN_DIR, "__init__.py")
    spec = importlib.util.spec_from_file_location(
        "wfrp_rag_plugin",
        init_path,
        submodule_search_locations=[PLUGIN_DIR],
    )
    if spec is None or spec.loader is None:
        return None
    mod = importlib.util.module_from_spec(spec)
    sys.modules["wfrp_rag_plugin"] = mod
    sys.modules["wfrp_rag_plugin"].__path__ = [PLUGIN_DIR]
    spec.loader.exec_module(mod)
    return mod


_wfrp_plugin = _load_plugin_module()
