import { useNavigate } from "react-router";

export function LoginButton() {
    const navigate = useNavigate();

    return (
        <button className="nav-btn" onClick={() => navigate("/login")}>
            Sign In
        </button>
    );
}
