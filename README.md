### OptiLog Artifacts

This repository contains the artifacts accompanying the paper titled:

```
OptiLog: Assigning Roles in Byzantine Consensus. Hanish Gogada, Christian Berger, Leander Jehl, Hans P Reiser and Hein Meling. EuroSys 2026.
```

OptiLog offers a comprehensive framework for recording experimental measurements and utilizing them to compute optimized system configurations.
This repository contains three primary artifacts utilized in the evaluation of OptiLog and its usecases.
OptiLog is accepted at [Eurosys2026](https://2026.eurosys.org/index.html). 

In addition to the contents of this repository, our evaluation also relies on artifacts from the following public repositories:
- [HotStuff](https://github.com/relab/hotstuff)
- [OptiAware](https://github.com/bergerch/opti-aware)

For the final artifact submission, these two repositories have been incorporated into this repository to ensure completeness and ease of use. 

### Overview
Each of the directories contains the README to explain the usage of the artifact.

- [Opti-AWARE](./opti-aware/README.md) explains the artifacts used to generate the results for plots figure 5 and 6 of the paper.
- [Optitree](./optitree/README.md) contains the code and scripts required to perform the experiments described in figures 9 and 12.
- [fig10_config](./fig10_config/README.md) contains the configuration file required to perform the experiments described in figure 10.
- [overhead](./overhead/README.md) contains the code to conduct the experiment described in figure 13.
- [wriggle](./wriggle/README.md) contains the code to perform the experiment explained in figure 11.


> **Note:** The Opti-AWARE component has a different license than the rest of this repository. Please refer to the license file in the [`opti-aware`](./opti-aware/LICENSE) directory for details.