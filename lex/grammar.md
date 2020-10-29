# grammar

    Assignment  => Expr AssignmentOp Expr
    Block       => { Expr ; ... }
    SumExpr     => Expr + Expr | Expr - Expr
    MultExpr    => Expr * Expr | Expr / Expr | Expr % Expr
    Invocation  => Expr ( Expr , ... )
    ParenExpr   => ( Expr )
