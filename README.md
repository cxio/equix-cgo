# equix

Go cgo 封装，绑定 [tevador/equix](https://github.com/tevador/equix) 的 **equix_v2** 分支（HashWX）。

    import "github.com/cxio/equix"

    sols, err := equix.SolveWithNonce(challenge, nonce)
    if err != nil {
        return err
    }
    for _, sol := range sols {
        h, err := equix.VerifyWithHashesAndNonce(challenge, nonce, sol)
        if err != nil {
            return err
        }
        _ = h
    }

需要 `CGO_ENABLED=1` 和 C 编译器（gcc/clang；Windows 用 MinGW）。不支持 `CGO_ENABLED=0`。

钉死的上游 commit：

- equix `350a85dedda1344637dac09a1de786ee63a5fb01`（`equix_v2`）
- hashx `08babdf4f41b0b8991d1fa94914c7c6902de0cb6`
- hashwx `d771cbf6cdc070755f7d137cdcf9d781af14da3f`

## 许可

本仓库 Go 代码为 MIT。`third_party/equix`、`third_party/hashx`、`third_party/hashwx` 为 LGPL-3.0。静态链入官方 C 构成 LGPL 组合作品；分发二进制时须保留 LGPL 声明，并提供这些 C 源码（本仓库已包含）。
