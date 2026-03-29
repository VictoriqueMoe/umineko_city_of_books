import {useNavigate} from "react-router";
import {Button} from "../../Button/Button";

export function LoginButton() {
    const navigate = useNavigate();

    return (
        <Button variant="primary" onClick={() => navigate("/login")}>
            Sign In
        </Button>
    );
}
